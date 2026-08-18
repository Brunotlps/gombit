package auth_test

import (
	"context"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/auth"
)

func TestHasPermission(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t)
	if err := auth.Migrate(db.DB); err != nil {
		t.Fatalf("auth.Migrate() error = %v", err)
	}

	superuser := auth.User{
		Email:        "superuser@example.com",
		PasswordHash: "unused",
		IsSuperuser:  true,
	}
	groupUser := auth.User{Email: "group@example.com", PasswordHash: "unused"}
	directUser := auth.User{Email: "direct@example.com", PasswordHash: "unused"}
	plainUser := auth.User{Email: "plain@example.com", PasswordHash: "unused"}
	for _, user := range []*auth.User{&superuser, &groupUser, &directUser, &plainUser} {
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Email, err)
		}
	}

	groupPermission, err := auth.EnsurePermission(ctx, db.DB, "admin.widgets.view", "View widgets")
	if err != nil {
		t.Fatalf("EnsurePermission(group): %v", err)
	}
	directPermission, err := auth.EnsurePermission(ctx, db.DB, "admin.widgets.create", "Create widgets")
	if err != nil {
		t.Fatalf("EnsurePermission(direct): %v", err)
	}
	viewers, err := auth.EnsureGroup(ctx, db.DB, "viewers")
	if err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	if err := auth.GrantPermissionToGroup(ctx, db.DB, &viewers, &groupPermission); err != nil {
		t.Fatalf("GrantPermissionToGroup: %v", err)
	}
	if err := auth.AddUserToGroup(ctx, db.DB, &groupUser, &viewers); err != nil {
		t.Fatalf("AddUserToGroup: %v", err)
	}
	if err := auth.GrantPermissionToUser(ctx, db.DB, &directUser, &directPermission); err != nil {
		t.Fatalf("GrantPermissionToUser: %v", err)
	}

	tests := []struct {
		name string
		user auth.User
		key  string
		want bool
	}{
		{name: "superuser bypass", user: superuser, key: "anything.at.all", want: true},
		{name: "group permission", user: groupUser, key: groupPermission.Key, want: true},
		{name: "direct permission", user: directUser, key: directPermission.Key, want: true},
		{name: "no permission", user: plainUser, key: groupPermission.Key, want: false},
		{name: "missing user", user: auth.User{ID: plainUser.ID + 10_000}, key: groupPermission.Key, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, checkErr := auth.HasPermission(ctx, db.DB, tt.user, tt.key)
			if checkErr != nil {
				t.Fatalf("HasPermission() error = %v", checkErr)
			}
			if got != tt.want {
				t.Fatalf("HasPermission(%q) = %t, want %t", tt.key, got, tt.want)
			}
		})
	}
}

func TestPermissionAssociationsPersist(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t)
	if err := auth.Migrate(db.DB); err != nil {
		t.Fatalf("auth.Migrate() error = %v", err)
	}

	user := auth.User{Email: "persisted@example.com", PasswordHash: "unused"}
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	permission, err := auth.EnsurePermission(ctx, db.DB, "admin.widgets.view", "View widgets")
	if err != nil {
		t.Fatalf("EnsurePermission: %v", err)
	}
	group, err := auth.EnsureGroup(ctx, db.DB, "persisted-viewers")
	if err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	if err := auth.GrantPermissionToGroup(ctx, db.DB, &group, &permission); err != nil {
		t.Fatalf("GrantPermissionToGroup: %v", err)
	}
	if err := auth.AddUserToGroup(ctx, db.DB, &user, &group); err != nil {
		t.Fatalf("AddUserToGroup: %v", err)
	}

	var loaded auth.User
	if err := db.WithContext(ctx).
		Preload("Groups.Permissions").
		First(&loaded, user.ID).Error; err != nil {
		t.Fatalf("reload user associations: %v", err)
	}
	if len(loaded.Groups) != 1 || len(loaded.Groups[0].Permissions) != 1 {
		t.Fatalf("persisted associations = %+v, want one group permission", loaded.Groups)
	}
}
