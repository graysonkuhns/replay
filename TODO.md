# To Do List

## v0.1.0

* Docs
  * Update embedded copy in command docs
* Integration testing
  * Move command
    * Test with --count flag
    * Test with additional payload types
    * Test that the message body is the same as expected
      * Plain text
      * Avro
      * Protobuf
      * YAML
  * DLR command
    * Test quit option
    * Test with --pretty-json flag
    * Test that the message body is the same as expected
    * Test that the correct messages were moved and the correct messages were discarded
    * Test with additional payload types
      * Plain text
      * Avro
      * Protobuf
      * YAML
* Refactoring
* Release and provide install instructions using homebrew
