# Test Environment

## Prerequisites

* [tfenv](https://github.com/tfutils/tfenv)
* A GCP project to deploy test resources in

## Setup

* Allow terraform to access your GCP project - `gcloud auth application-default login`
* Install terraform - `tfenv install` (version from .terraform-version file will be used)
* Create terraform variables file - `cp terraform.tfvars.template terraform.tfvars`
* Update values in `terraform.tfvars` file
* Install terraform modules - `terraform init`
