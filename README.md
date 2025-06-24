# ModelFuzz-Java

ModelFuzz is a fuzzer developed for fuzzing distributed systems. This repository supports testing Ratis and Xraft. 

In the parent folder of this repository, please clone:

`https://github.com/egeberkaygulcan/xraft-controlled`

`https://github.com/egeberkaygulcan/ratis-fuzzing`

`https://github.com/burcuku/tlc-controlled-with-benchmarks`

## TLC
Within the `tlc-controlled-with-benchmarks` repository folder, first replace the tlc-controlled folder with the clone of:

`https://github.com/burcuku/tlc-controlled`

To compile the tlc controlled repository: 

``` shell
ant -f customBuild.xml compile
```

(Optional) Compiling test classes:

``` shell
ant -f customBuild.xml compile-test
```

Build the tlatool.jar file:

``` shell
ant -f customBuild.xml dist
```

To run TLC, within the tlc-controlled fodler:

``` shell
java -jar dist/tla2tools_server.jar -controlled <path-to-tla-file> -config <path-to-cfg-file> -mapperparams "name=raft"
```

The *.tla* and *.cfg* files can be found within the *tla-benchmarks* folder. Please use the models in the *Raft* folder. Additionally, please change `raft_alt` in the **EXTENDS** part of the *.tla* files in to `raft-enhanced` if it is not already changed.

## Ratis
To build Ratis (within the Ratis repository folder), you can use: 

``` shell
mvn clean package -DskipTests
```

## Xraft
To build Xraft (within the Xraft repository folder):

``` shell
mvn clean compile install
```

``` shell
cd xraft-kvstore
```

``` shell
mvn package assembly:single
```

## ModelFuzz usage

You can use this repository by building and running the executable:

``` shell
go build
```

``` shell
./modelfuzz-java
```

Example configuration is given in the *main.go* file

# Additions of Martijn and Shantanu
## Mutation strategy
 
In the `main.go` file, you can select a strategy from three options: `CodeAndStateCoverage`, `StateCoverage`, and `TransitionCoverage`. This choice determines the number of mutations the fuzzer generates during execution. You can cap the number of mutations in the `main.go` file, by adjusting the `maxMutations` parameter.

## Saving output

After an experiment finishes, the user is prompted to specify the name of the directory where the output should be saved. The output is always stored inside the `finalOutputs` folder, within a subdirectory corresponding to the chosen mutation strategy. The exact name of this subdirectory is determined by the user’s input.

## Visualisation
To visualize the data, there are two scripts available in the `scripts` directory.

- Use the `createPlots.py` script to generate plots from your experiment data. To prepare for this, copy all relevant experiment output data into a single directory named `finalOutputs`. Inside `finalOutputs`, each subdirectory should contain the original output data from one or more experiments. When you run `createPlots.py`, it will process every subdirectory within `finalOutputs` and create a new directory named `graphs` inside each one. This `graphs` directory will contain all the generated plots based on the output data found in that specific subdirectory.

- Use the `plotLive.py` script during an experiment to visualize data in real time. When run, this Python program will read the data being saved in the `output` directory of the running experiment and generate live plots based on that data.


