module "remote_module" {
  source = "git::https://github.com/im2nguyen/rover.git//example/random-test/random-name"

  max_length = "3"

}