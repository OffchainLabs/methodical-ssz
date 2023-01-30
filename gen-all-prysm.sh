#! /bin/bash

# bad github.com/prysmaticlabs/prysm/v3/proto/engine/v1
#dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient github.com/kasey/methodical-ssz/cmd/ssz -- gen --type-names=ExecutionPayload,ExecutionPayloadCapella github.com/prysmaticlabs/prysm/v3/proto/engine/v1
go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=ExecutionPayload,ExecutionPayloadCapella github.com/prysmaticlabs/prysm/v3/proto/engine/v1
# good github.com/prysmaticlabs/prysm/v3/proto/engine/v1
# go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=ExecutionPayloadHeader,ExecutionPayloadHeaderCapella,Withdrawal github.com/prysmaticlabs/prysm/v3/proto/engine/v1

# good github.com/prysmaticlabs/prysm/v3/proto/eth/v1
#go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=Attestation,AggregateAttestationAndProof,SignedAggregateAttestationAndProof,AttestationData,Checkpoint,BeaconBlock,SignedBeaconBlock,BeaconBlockBody,ProposerSlashing,AttesterSlashing,Deposit,VoluntaryExit,SignedVoluntaryExit,Eth1Data,BeaconBlockHeader,SignedBeaconBlockHeader,IndexedAttestation,SyncAggregate,Deposit_Data,Validator github.com/prysmaticlabs/prysm/v3/proto/eth/v1

# bad github.com/prysmaticlabs/prysm/v3/proto/eth/v2
 #go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=SignedBeaconBlockBellatrix,SignedBeaconBlockCapella,BeaconBlockBellatrix,BeaconBlockCapella,BeaconBlockBodyBellatrix,BeaconBlockBodyCapella github.com/prysmaticlabs/prysm/v3/proto/eth/v2
# good github.com/prysmaticlabs/prysm/v3/proto/eth/v2
# go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=SignedBlindedBeaconBlockBellatrix,SignedBlindedBeaconBlockCapella,SignedBeaconBlockAltair,BlindedBeaconBlockBellatrix,BlindedBeaconBlockCapella,BeaconBlockAltair,BlindedBeaconBlockBodyBellatrix,BlindedBeaconBlockBodyCapella,BeaconBlockBodyAltair,BLSToExecutionChange,SignedBLSToExecutionChange github.com/prysmaticlabs/prysm/v3/proto/eth/v2

# bad github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
 #go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=SignedBeaconBlockBellatrix,BeaconBlockBellatrix,BeaconBlockBodyBellatrix,SignedBeaconBlockCapella,BeaconBlockCapella,BeaconBlockBodyCapella,BuilderBidCapella,HistoricalSummary github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
# good github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
# go run github.com/kasey/methodical-ssz/cmd/ssz gen --type-names=Attestation,AggregateAttestationAndProof,SignedAggregateAttestationAndProof,AttestationData,Checkpoint,BeaconBlock,SignedBeaconBlock,BeaconBlockAltair,SignedBeaconBlockAltair,BeaconBlockBody,BeaconBlockBodyAltair,ProposerSlashing,AttesterSlashing,Deposit,VoluntaryExit,SignedVoluntaryExit,Eth1Data,BeaconBlockHeader,SignedBeaconBlockHeader,IndexedAttestation,SyncAggregate,SignedBlindedBeaconBlockBellatrix,BlindedBeaconBlockBellatrix,BlindedBeaconBlockBodyBellatrix,SignedBlindedBeaconBlockCapella,BlindedBeaconBlockCapella,BlindedBeaconBlockBodyCapella,ValidatorRegistrationV1,SignedValidatorRegistrationV1,BuilderBid,Deposit_Data,BeaconState,BeaconStateAltair,Fork,PendingAttestation,HistoricalBatch,SigningData,ForkData,DepositMessage,SyncCommittee,SyncAggregatorSelectionData,BeaconStateBellatrix,BeaconStateCapella,PowBlock,Status,BeaconBlocksByRangeRequest,ENRForkID,MetaDataV0,MetaDataV1,SyncCommitteeMessage,SyncCommitteeContribution,ContributionAndProof,SignedContributionAndProof,Validator,BLSToExecutionChange,SignedBLSToExecutionChange github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
