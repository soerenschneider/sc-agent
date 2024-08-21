terraform {
  required_version = ">= 1.6.0"
  required_providers {
    vault = {
      source  = "hashicorp/vault"
      version = "4.2.0"
    }
  }
}

provider "vault" {
  address = "http://localhost:8200"
  token   = "test"
}
