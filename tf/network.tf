
resource "google_compute_backend_service" "backend_service" {
  name                  = "backend-service"
  health_checks                   = [google_compute_health_check.compute_health_check.id]
  load_balancing_scheme           = "EXTERNAL_MANAGED"
  port_name                       = "http"
  protocol                        = "HTTP"
  session_affinity                = "NONE"
  timeout_sec                     = 30
  backend {
    group           = google_compute_region_instance_group_manager.instance_group_manager.instance_group
    balancing_mode  = "UTILIZATION"
    capacity_scaler = 1.0
  }
}

resource "google_compute_health_check" "compute_health_check" {
  name               = "basic-check"
  check_interval_sec = 5
  healthy_threshold  = 2
  http_health_check {
    port               = 8080
    port_specification = "USE_FIXED_PORT"
    request_path       = "/"
  }
  timeout_sec         = 5
  unhealthy_threshold = 2
}

resource "google_compute_url_map" "url_map" {
  name            = "url-map"
  default_service = google_compute_backend_service.backend_service.self_link
}

resource "google_compute_managed_ssl_certificate" "ssl_certificate" {
  name = "managed-cert"

  managed {
    domains = ["phantom.hagaley.com"]
  }
}

resource "google_compute_target_https_proxy" "compute_target_https_proxy" {
  name     = "https-proxy"
  url_map  = google_compute_url_map.url_map.id
  ssl_certificates = [
    google_compute_managed_ssl_certificate.ssl_certificate.name
  ]
  depends_on = [
    google_compute_managed_ssl_certificate.ssl_certificate
  ]
}

resource "google_compute_global_address" "compute_global_address" {
  name = "global-address"
}

resource "google_compute_global_forwarding_rule" "compute_global_forwarding_rule" {
  name                  = "https-content-rule"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "443"
  target                = google_compute_target_https_proxy.compute_target_https_proxy.id
  ip_address            = google_compute_global_address.compute_global_address.address
}
