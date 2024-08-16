variable "SHA" {
  description = "The SHA of the image to deploy"
}

provider "google" {
  project = "jahagaley"
  region  = "us-west1"
}

module "gce-container" {
  source = "terraform-google-modules/container-vm/google"
  version = "~> 3.1"

  container = {
    image="us-docker.pkg.dev/jahagaley/phantom:main.${var.SHA}"
    tty : true

    # Declare volumes to be mounted.
    # This is similar to how docker volumes are declared.
    volumeMounts = []
  }

  restart_policy = "Always"
}

resource "google_compute_instance_template" "instance_template" {
  name                    = "instance-template-${substr(var.SHA, 0, 8)}"
  machine_type            = "e2-micro"
  tags                    = ["http-server", "https-server", "allow-health-check"]

  disk {
    source_image = "cos-cloud/cos-stable"
  }

  network_interface {
    network = "default"
    access_config {
      // Ephemeral IP
    }
  }

  service_account {
    email  = "phantom@jahagaley.iam.gserviceaccount.com"
    scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }

  metadata = {
    "gce-container-declaration" = module.gce-container.metadata_value
    "google-logging-enabled"    = "true"
    "google-monitoring-enabled" = "true"
  }

  labels = {
    "container-vm" = module.gce-container.vm_container_label
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_region_instance_group_manager" "instance_group_manager" {
  name               = "instance-group-manager"
  base_instance_name = "instance"
  region               = "us-west1"
  target_size        = 1 // Set initial size, auto-scaling will adjust this.

  version {
    instance_template  = google_compute_instance_template.instance_template.self_link_unique
  }

  named_port {
    name = "http"
    port = 8080
  }

  update_policy {
    type = "PROACTIVE"
    minimal_action = "REPLACE"
  }
}
