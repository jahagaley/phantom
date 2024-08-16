# Phantom

Phantom is a CI/CD solution built on Google Cloud. This service is offers
a simple way to build code, run tests, and deploy to different applications
on different GCP services (Cloud Run, GKE, GCE, etc).

Phantom alo handles the end to end lifecycle for applications such as:
- Builds and tests on every commit
- Cleanup of unused images (ex: a branch has been merged).
- Tracking which revision for an application is deployed.
