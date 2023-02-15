methodical-ssz
--------------

This tool generates code that uses the [prysm fork](https://github.com/prysmaticlabs/fastssz/) of the [fastssz api](https://github.com/ferranbt/fastssz/) (with bindings for [go-hashtree](https://github.com/prysmaticlabs/gohashtree) coming very soon!), for marshaling, unmarshaling and merkleization of go types.

Generate ssz methodsets
-----------------------

 The source types are parsed using `go/types`, which locates the source code for a given package path through go's local package discovery utilities, so a current limitation is that you need to fetch the package to your go package tree before using the tool. This can be done with go get:
```
go get github.com/prysmaticlabs/prysm/v3@v3.2.1
```

Once this is done, code generation can be run against go types in the desired package:
```
go run ./cmd/ssz gen --output=beacon-state.ssz.go --type-names BeaconState,BeaconStateAltair,BeaconStateBellatrix github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
Generating methods for github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/BeaconState
Generating methods for github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/BeaconStateAltair
Generating methods for github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/BeaconStateBellatrix
```

The generated file can then be placed in a local directory for the source package so that the generated methods become part of the methodset of the source type.

Generate spectests for a package
--------------------------------

This tool has a subcommand to generate go tests from the `ssz_static` tests in the ethereum/consensus-spec-tests repository. The generated tests can be run as normal go tests, which is helpful in identifying specific failure cases and walking through generated code with a debugger.

spectest tarball
================

To get the spec test fixtures for codegen, go to the [consensus-spec-test repository releases page](https://github.com/ethereum/consensus-spec-tests/releases) and download the most recent `pre-release` or `latest` spec tests release.
```
curl -L https://github.com/ethereum/consensus-spec-tests/releases/download/v1.3.0-rc.2-hotfix/mainnet.tar.gz > mainnet-v1.3.0-rc.2.tar.gz
```

yaml config
===========

The spectest generation tool needs a mapping between go types and consensus spec container types. This mapping is described as a yaml config file. The mappings for prysm types are committed to the repo in a go `testdata` fixture directory, in `specs/testdata/prysm.yaml`. The following snippet illustrates the format:
```
package: github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
preset: mainnet
defs:
  - fork: phase0
    types:
      - name: AggregateAndProof
        type_name: AggregateAttestationAndProof
      - name: Attestation
      # ...
      - name: BeaconBlock
      # ...
  - fork: altair
    types:
      - name: BeaconBlock
        type_name: BeaconBlockAltair
      - name: BeaconBlockBody
        type_name: BeaconBlockBodyAltair
```
`package` is the go package containing the types for code generation. Each yaml holds the description for one go package. `preset` is one of mainnet or minimal. At this time only mainnet is supported as preset customization is specific to an implementation's build tooling. `defs` contains a list of yaml objects, with the `fork` key matching the fork directory in the spectest layout, and the `types` key containing a list of objects connecting the `name` of the type from consensus specs / spectest directory names, to the `type_name` found in the `package`. If `type_name` is not present, `name` is the default, so for go types with the same spelling and capitalization as the consensus type, the `name` field alone is all that's needed to specify the type.

The config reader processes type mappings in canonical fork order, so if a type's schema has not changed since the previous fork, it does not need to be redeclared. For instance the mapping for `Attestation` is only described once in the `phase0` mapping; the same go type will be used to execute the spectest for the Attestation value for all subsequent forks. `BeaconBlock`, on the other hand, has been redefined at every fork, so tests will use `BeaconBlockAltair` for spectests in the `altair` tree and so on. Any types observed in the tarball that don't have an entry defined in the config yaml will be skipped with a warning.

spectest subcommand
===================

The `generated` directory is a good place to stick generated tests as it is already in the .gitignore for the project. Assuming the tarball in the example above has been downloaded to the repo directory, running the following command there will generate spectests for all prysm types described in the test fixture yaml config:
```
go run ./cmd/ssz spectest --release-uri=file://$PWD/mainnet-v1.3.0-rc.2.tar.gz --config=$PWD/specs/testdata/prysm.yaml --output=$PWD/generated
```

Run the spectest like normal go tests:
```
go test ./generated
ok  	github.com/OffchainLabs/methodical-ssz/generated	1.003s
```
