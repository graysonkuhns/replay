terraform {
  required_version = ">= 0.12"

  # Use the local backend to store state in a file.
  backend "local" {
    path = "terraform.tfstate"
  }
}

variable "gcp_project" {
  description = "The GCP project to deploy resources to"
  type        = string
}

variable "gcp_region" {
  description = "The GCP region to deploy resources to"
  type        = string
}

provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
}

# Use the current workspace name as the environment
locals {
  environment = terraform.workspace
}
