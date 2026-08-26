"""Django's test runner creates and migrates a real, throwaway PostgreSQL
database (test_<DATABASES["default"]["NAME"]>) for `manage.py test` — these
tests need no separate -database.dsn flag or docker-compose step the way
the Go apps' //go:build integration suites do; `manage.py test` against a
live Postgres (DATABASE_URL pointing at any reachable instance, e.g.
benchmarks/compose.yml's postgres service) is sufficient.

Covers the same contract benchmarks/apps/gin-gorm/main_test.go and
benchmarks/apps/gombit/internal/project/handler_test.go pin for their own
implementations (benchmarks/docs/schema.md "Canonical CRUD API"), so the
three Go/Python suites can't silently diverge from each other while each
claims to satisfy the same contract.
"""
import json

from django.test import Client, TestCase, TransactionTestCase, override_settings

from .management.commands.seed import (
    project_description,
    project_name,
    project_owner_id,
    seed_database_n,
    user_email,
    user_name,
)
from .models import Project, User

# Django's test client sends Host: testserver, which CommonMiddleware
# rejects unless it's in ALLOWED_HOSTS — settings.py's default
# (127.0.0.1,localhost) is right for gunicorn but not for this in-process
# client.
_client = Client()


@override_settings(ALLOWED_HOSTS=["testserver"])
class ProjectCRUDTests(TestCase):
    def setUp(self):
        self.owner = User.objects.create(email="owner@example.com", name="Owner")

    def _post(self, body):
        return _client.post(
            "/api/projects", data=json.dumps(body), content_type="application/json"
        )

    def _patch(self, pk, body):
        return _client.patch(
            f"/api/projects/{pk}", data=json.dumps(body), content_type="application/json"
        )

    def test_crud_round_trip(self):
        create = self._post({"owner_id": self.owner.pk, "name": "Test Project", "description": "desc"})
        self.assertEqual(create.status_code, 201, create.content)
        created = create.json()["data"]
        self.assertEqual(created["name"], "Test Project")
        self.assertEqual(created["owner_name"], "Owner")

        get = _client.get(f"/api/projects/{created['id']}")
        self.assertEqual(get.status_code, 200)

        patch = self._patch(created["id"], {"name": "Renamed"})
        self.assertEqual(patch.status_code, 200, patch.content)
        updated = patch.json()["data"]
        self.assertEqual(updated["name"], "Renamed")
        self.assertEqual(updated["description"], "desc")  # unchanged

        delete = _client.delete(f"/api/projects/{created['id']}")
        self.assertEqual(delete.status_code, 200)

        get_after_delete = _client.get(f"/api/projects/{created['id']}")
        self.assertEqual(get_after_delete.status_code, 404)

    def test_create_rejects_blank_name(self):
        for name in ("", "   "):
            with self.subTest(name=name):
                response = self._post({"owner_id": self.owner.pk, "name": name, "description": "x"})
                self.assertEqual(response.status_code, 422, response.content)

    def test_update_rejects_blank_name(self):
        create = self._post({"owner_id": self.owner.pk, "name": "Original", "description": "desc"})
        project_id = create.json()["data"]["id"]

        for name in ("", "   "):
            with self.subTest(name=name):
                response = self._patch(project_id, {"name": name})
                self.assertEqual(response.status_code, 422, response.content)

        # A rejected update must not have partially applied.
        unchanged = _client.get(f"/api/projects/{project_id}").json()["data"]
        self.assertEqual(unchanged["name"], "Original")

    def test_create_rejects_zero_owner_id(self):
        response = self._post({"owner_id": 0, "name": "x", "description": "y"})
        self.assertEqual(response.status_code, 422, response.content)

    def test_create_rejects_nonexistent_owner_id(self):
        """Unlike benchmarks/apps/gombit (which documents a real, deliberately
        unfixed Gombit framework gap that turns this same case into a 500),
        this from-scratch Django app gets it right from the start —
        matching benchmarks/apps/gin-gorm's TestCreateRejectsInvalidOwnerID,
        the correct behavior every implementation not carrying that
        specific Gombit gap is expected to have.
        """
        response = self._post({"owner_id": 999999, "name": "Orphan", "description": "no such owner"})
        self.assertEqual(response.status_code, 422, response.content)

    def test_get_nonexistent_id_is_not_found(self):
        response = _client.get("/api/projects/999999999")
        self.assertEqual(response.status_code, 404)

    def test_get_non_numeric_id_is_not_found(self):
        """A route parameter that isn't an id at all must still get the D10
        not_found envelope, not Django's own routing-level plain-text 404 —
        see urls.py's <str:pk> comment.
        """
        response = _client.get("/api/projects/not-a-number")
        self.assertEqual(response.status_code, 404)
        self.assertIn("error", response.json())


@override_settings(ALLOWED_HOSTS=["testserver"])
class ProjectListTests(TestCase):
    def _seed_fixture(self, user_count, project_count):
        """Inserts users/projects directly through the ORM, independent of
        seed.py's full-scale seed_database_n, so list tests run in
        milliseconds — mirrors benchmarks/apps/gin-gorm's seedFixture.

        Round-robins over the *actual* pks bulk_create returns, not
        project_owner_id(i, user_count)'s assumed 1..user_count range:
        TestCase isolates each test's rows via a rolled-back transaction,
        but Postgres sequences are not transactional (nextval() survives a
        rollback), so a fixture User created in an earlier test can push
        this test's own users past pk 1..user_count. gin-gorm/gombit don't
        hit this because their equivalent helper runs right after a fresh
        TRUNCATE ... RESTART IDENTITY, which this in-transaction test
        deliberately avoids (see SeedDatabaseIsIdempotentAndCorrectTests'
        TransactionTestCase for why TRUNCATE can't run here).
        """
        users = User.objects.bulk_create(
            [User(email=f"fixture-{i}@example.com", name=f"Fixture User {i}") for i in range(1, user_count + 1)]
        )
        Project.objects.bulk_create(
            [
                Project(
                    owner_id=users[project_owner_id(i, user_count) - 1].pk,
                    name=f"Fixture Project {i}",
                    description="fixture",
                )
                for i in range(1, project_count + 1)
            ]
        )
        return users

    def test_list_pagination_and_ordering(self):
        self._seed_fixture(3, 25)

        page1 = _client.get("/api/projects?page=1&limit=20").json()
        self.assertEqual(page1["meta"], {"page": 1, "limit": 20, "total": 25})
        self.assertEqual(len(page1["data"]), 20)
        self.assertGreater(page1["data"][0]["id"], page1["data"][1]["id"])
        for row in page1["data"]:
            self.assertNotEqual(row["owner_name"], "")

        page2 = _client.get("/api/projects?page=2&limit=20").json()
        self.assertEqual(len(page2["data"]), 5)
        self.assertGreater(page1["data"][19]["id"], page2["data"][0]["id"])

    def test_list_does_not_n_plus_1(self):
        """benchmarks/docs/schema.md "Canonical CRUD API": the list endpoint
        must issue a fixed number of queries independent of page size, not
        one owner query per row. select_related("owner") makes this a
        single JOIN — 2 queries total (COUNT + the joined page SELECT),
        for both a non-empty and an empty page (views.py's doc comment
        explains why this is 2 rather than gin-gorm's 3, and why schema.md
        permits it).
        """
        self._seed_fixture(5, 20)
        with self.assertNumQueries(2):
            response = _client.get("/api/projects?page=1&limit=20")
        self.assertEqual(response.status_code, 200)

    def test_list_does_not_n_plus_1_empty_page(self):
        self._seed_fixture(5, 20)
        with self.assertNumQueries(2):
            response = _client.get("/api/projects?page=99&limit=20")
        self.assertEqual(response.status_code, 200)


class SeedContentTests(TestCase):
    """Pure, no-database checks of the seed content formulas — port of
    benchmarks/apps/shared/seed_test.go's
    TestSeedContentIsDeterministic/TestProjectOwnerIDRoundRobin. This app
    can't import that Go package, so the same properties are re-verified
    against this app's own from-scratch port instead of merely trusted.
    """

    def test_seed_content_is_deterministic(self):
        self.assertEqual(user_email(1), "user-0001@example.com")
        self.assertEqual(user_name(1), "User 0001")
        self.assertEqual(user_email(1000), "user-1000@example.com")
        self.assertEqual(project_name(1), "Project 000001")
        self.assertEqual(project_description(1), "Seeded benchmark project 000001")
        self.assertEqual(project_name(100000), "Project 100000")

    def test_project_owner_id_round_robin(self):
        user_count = 7
        self.assertEqual(project_owner_id(1, user_count), 1)
        self.assertEqual(project_owner_id(7, user_count), 7)
        # The 8th project wraps back to owner 1 — the round-robin boundary
        # an off-by-one would break silently.
        self.assertEqual(project_owner_id(8, user_count), 1)
        self.assertEqual(project_owner_id(14, user_count), 7)
        self.assertEqual(project_owner_id(15, user_count), 1)


class SeedDatabaseIsIdempotentAndCorrectTests(TransactionTestCase):
    """Mirrors benchmarks/apps/gin-gorm and benchmarks/apps/gombit's
    TestSeedDatabaseNIsIdempotentAndCorrect: exercises the real
    truncate-then-seed path at a small scale, twice, to confirm it
    truncates rather than accumulates duplicate data on repeated
    invocations.

    TransactionTestCase, not TestCase: plain TestCase wraps the whole test
    method in one outer transaction, and Postgres refuses to TRUNCATE a
    table that still has this transaction's own pending deferred
    foreign-key trigger events (projects.owner_id is DEFERRABLE INITIALLY
    DEFERRED — Django's own default for Postgres FKs) — calling
    seed_database_n twice in the same transaction hits exactly that.
    TransactionTestCase commits between operations instead of wrapping them
    in a savepoint, so each seed_database_n() call's TRUNCATE runs against
    a fully committed prior state, the same as it would in production
    (where each `manage.py seed` invocation is its own fresh
    connection/transaction to begin with).
    """

    def test_seed_database_n_is_idempotent_and_correct(self):
        user_count, project_count = 7, 23  # not a multiple, to exercise the round-robin remainder

        for run in (1, 2):
            seed_database_n(user_count, project_count)

            self.assertEqual(User.objects.count(), user_count, f"run {run}")
            self.assertEqual(Project.objects.count(), project_count, f"run {run}")

            first_user = User.objects.get(pk=1)
            self.assertEqual(first_user.email, user_email(1))
            self.assertEqual(first_user.name, user_name(1))

            # Project user_count+1 (8th project, user_count=7) is the first
            # to wrap back to owner 1 — the round-robin boundary a naive
            # off-by-one would break silently.
            wrapped = Project.objects.get(pk=user_count + 1)
            self.assertEqual(wrapped.owner_id, 1, f"run {run}")
            self.assertEqual(wrapped.name, project_name(user_count + 1), f"run {run}")
