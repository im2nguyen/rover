
variable "max_length" {
  default = 5
}

resource "random_integer" "pet_length" {
  min = 1
  max = var.max_length
}

resource "random_pet" "pet" {
  length = random_integer.pet_length.result
}

output "random_name" {
  value = random_pet.pet.id
}