terraform {
  required_version = ">= 1.1.0"

  required_providers {
    litellm = {
      source = "terraform.articulate.team/articulate/litellm"
    }
  }
}

provider "litellm" {
  api_base = var.litellm_api_base
  api_key  = var.litellm_api_key
}
