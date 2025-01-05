variable "GITHUB_ACCESS_TOKEN" {
    default = ""
}

variable "TAG" {
    default = "latest"
}

// Default target group that will be built when no specific target is specified
group "default" {
    targets = ["elder-wrap"]
}

// Base target with common settings
target "docker-metadata-action" {
    tags = ["elder-wrap:${TAG}"]
}

// Main build target
target "elder-wrap" {
    inherits = ["docker-metadata-action"]
    context = "."
    dockerfile = "Dockerfile"
    args = {
        GITHUB_ACCESS_TOKEN = "${GITHUB_ACCESS_TOKEN}"
    }
} 
