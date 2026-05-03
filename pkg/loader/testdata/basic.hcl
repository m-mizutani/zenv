DB_HOST = "localhost"
DB_USER = "admin"
GH_REPO = "ubie-inc/foo"

DB_PASS {
  value  = "secret"
  secret = true
}

SSL_CERT {
  file = "config_content.txt"
}

GIT_SHA {
  command = ["echo", "abc123"]
}

APP_HOME {
  alias = "ZENV_TEST_HOME"
}
