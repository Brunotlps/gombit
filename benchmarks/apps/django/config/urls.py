from django.http import HttpResponse
from django.urls import include, path


def livez(_request):
    return HttpResponse(status=200)


urlpatterns = [
    path("livez", livez),
    path("api/", include("projects.urls")),
]
