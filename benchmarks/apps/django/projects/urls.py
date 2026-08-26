from django.urls import path

from . import views

# <str:pk>, not <int:pk>: a non-numeric id must still reach the view and
# get the D10 not_found envelope (views._parse_pk), the same as
# benchmarks/apps/gin-gorm's strconv.ParseUint failure path — Django's
# <int:pk> converter would instead reject it at routing time with its own
# plain-text 404, a different response shape than every other
# implementation.
urlpatterns = [
    path("projects", views.ProjectListCreateView.as_view()),
    path("projects/<str:pk>", views.ProjectDetailView.as_view()),
]
