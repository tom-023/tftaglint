resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"

  tags = {
    Name        = "WebServer"
    Environment = "production"
    Owner       = "DevOps"
    Project     = "MyApp"
    ManagedBy   = "Terraform"
  }
}

resource "aws_instance" "db" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.medium"

  tags = {
    Name        = "Database"
    Environment = "production"
    Test        = "true"  # This should trigger a violation
    Project     = "MyApp"
  }
}

resource "aws_s3_bucket" "logs" {
  bucket = "my-app-logs"

  tags = {
    Environment = "development"
    project     = "MyApp"  # Wrong case - should be Project
  }
}

resource "aws_instance" "test" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.nano"

  tags = {
    Name        = "TestInstance"
    Environment = "invalid-env"  # Invalid environment value
    Project     = "TestProject"
    ManagedBy   = "Manual"
  }
}