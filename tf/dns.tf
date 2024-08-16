resource "google_dns_managed_zone" "dns_zone" {
  name        = "dns-zone"
  dns_name    = "phantom.hagaley.com"
}

resource "google_dns_record_set" "dns_record_https" {
  name         = google_dns_managed_zone.dns_zone.dns_name
  managed_zone = google_dns_managed_zone.dns_zone.name
  type    = "A"
  ttl     = 300
  rrdatas = [google_compute_global_forwarding_rule.compute_global_forwarding_rule.ip_address]
}
