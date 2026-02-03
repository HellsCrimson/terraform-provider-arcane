terraform {
  required_version = ">= 1.4.0"
  required_providers {
    arcane = {
      source  = "hellscrimson/arcane"
      version = ">= 0.0.1"
    }
  }
}

provider "arcane" {
  api_key  = var.arcane_api_key
  endpoint = var.arcane_endpoint
}

variable "arcane_api_key" {
  type      = string
  sensitive = true
}

variable "arcane_endpoint" {
  type    = string
  default = "http://localhost:3552/api"
}

# Create a WordPress template
resource "arcane_template" "wordpress" {
  name        = "WordPress Stack"
  description = "WordPress with MySQL database"

  content = <<-EOT
    version: '3.8'
    services:
      wordpress:
        image: wordpress:latest
        ports:
          - "8080:80"
        environment:
          WORDPRESS_DB_HOST: db
          WORDPRESS_DB_NAME: wordpress
          WORDPRESS_DB_USER: wordpress
          WORDPRESS_DB_PASSWORD: $${DB_PASSWORD}
        depends_on:
          - db
        volumes:
          - wordpress-data:/var/www/html

      db:
        image: mysql:5.7
        environment:
          MYSQL_DATABASE: wordpress
          MYSQL_USER: wordpress
          MYSQL_PASSWORD: $${DB_PASSWORD}
          MYSQL_ROOT_PASSWORD: $${ROOT_PASSWORD}
        volumes:
          - db-data:/var/lib/mysql

    volumes:
      wordpress-data:
      db-data:
  EOT

  env_content = <<-EOT
    DB_PASSWORD=changeme123
    ROOT_PASSWORD=rootpassword123
  EOT
}

# Create a simple Nginx template
resource "arcane_template" "nginx" {
  name        = "Nginx Web Server"
  description = "Simple Nginx web server template"

  content = <<-EOT
    version: '3.8'
    services:
      web:
        image: nginx:latest
        ports:
          - "$${PORT}:80"
        volumes:
          - ./html:/usr/share/nginx/html:ro
  EOT

  env_content = <<-EOT
    PORT=8080
  EOT
}

output "wordpress_template_id" {
  value = arcane_template.wordpress.id
}

output "nginx_template_id" {
  value = arcane_template.nginx.id
}
