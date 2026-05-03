API_URL {
  value = "https://api.example.com"
  profile {
    dev     = "http://localhost:8080"
    staging = "https://staging.api.example.com"
  }
}

DEBUG_MODE {
  value = "false"
  profile {
    dev  = "true"
    prod = null
  }
}

SSL_CERT {
  file = "config_content.txt"
  profile {
    dev {
      value = "dev-cert-content"
    }
  }
}
