# modelfuzz-java

ModelFuzz is a fuzzer developed for fuzzing distributed systems. This repository supports testing Ratis and Xraft. 

In the parent folder of this repository you should clone:

`https://github.com/egeberkaygulcan/xraft-controlled`

`https://github.com/egeberkaygulcan/ratis-fuzzing`

To build Ratis (within the Ratis repository folder), you can use: 

`mvn clean package -DskipTests`

To build Xraft (within the Xraft repository folder):

`mvn clean compile install`

`cd xraft-kvstore`

`mvn package assembly:single`

## ModelFuzz usage

You can use this repository by building and running the executable:

`go build`

`./modelfuzz-java`

Example configuration is given in the *main.go* file

### Parameters

-TODO

