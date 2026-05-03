DB_USER = "admin"
DB_HOST = "localhost"
DB_NAME = "myapp"

DATABASE_URL {
  value = "postgresql://{{ .DB_USER }}@{{ .DB_HOST }}/{{ .DB_NAME }}"
  refs  = ["DB_USER", "DB_HOST", "DB_NAME"]
}

USE_STAGING = "true"
API_ENDPOINT {
  value = "{{ if eq .USE_STAGING \"true\" }}https://staging.api.example.com{{ else }}https://api.example.com{{ end }}"
  refs  = ["USE_STAGING"]
}
