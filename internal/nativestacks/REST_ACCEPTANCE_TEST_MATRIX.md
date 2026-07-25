# REST acceptance test matrix

This maps every acceptance criterion in `GITHUB_STACKS_REST_API.md` section 13
to automated evidence. The API remains private preview; this matrix covers the
fixture and orchestration contract, not the separate live-validation items.

| # | Acceptance criterion | Automated evidence |
| --- | --- | --- |
| 1 | Stacked and standalone membership decoding | `TestRESTModelsDecodePublishedShapesAndUnknownFields` |
| 2 | Direct PR base versus ultimate Stack base | `TestRESTModelsDecodePublishedShapesAndUnknownFields` |
| 3 | Stack decoding with unknown fields | `TestRESTModelsDecodePublishedShapesAndUnknownFields` |
| 4 | Bottom-to-top order preservation | `TestRESTModelsDecodePublishedShapesAndUnknownFields`, `TestClassify` |
| 5 | Merged versus closed-unmerged | `TestStackMemberMergedAtDistinguishesClosedOutcomes`, `TestEnsureUnstackAllowsCleanup` |
| 6 | Multi-page listing and filtered lookup | `TestListStacksUsesPaginationAndFlattensPages`, `TestFindStackForPRAcceptsZeroOneAndRejectsAmbiguous` |
| 7 | Create boundaries 2 and 100 | `TestCreateAndAppendRequestBoundaries`, `TestCreateAndAppendAcceptMaximumRequestSize` |
| 8 | Append boundaries 1 and 100 | `TestCreateAndAppendRequestBoundaries`, `TestCreateAndAppendAcceptMaximumRequestSize` |
| 9 | Valid and broken base/head chains | `TestValidateWritePlanChecksChainRepositoryAndLifecycle` |
| 10 | Create, no-op, and append-prefix reconciliation | `TestClassify`, `TestReconcileNativeStackUsesDirectRESTClient` |
| 11 | Remote-extra, reordered, mixed, and stacked-suffix conflicts | `TestClassify` |
| 12 | Create 201 and append 200 verification | `TestCreateAndAppendUseDocumentedPayloadsAndStatuses` |
| 13 | Partial unstack 200 | `TestUnstackDistinguishesPartialAndDissolved`, `TestEnsureUnstackAllowsCleanup` |
| 14 | Dissolved unstack 204 | `TestUnstackDistinguishesPartialAndDissolved` |
| 15 | Ambiguous 404 classification | `TestProbeAvailabilityDisambiguatesRepositoryLevel404`, `TestNumberedStack404IsNotFeatureUnavailable` |
| 16 | Validation 422 without blind retry | `TestValidationFailureIsNotRetried` |
| 17 | Transport failure followed by reconciliation | `TestUncertainWritesReconcileBeforeReturning` |
| 18 | Completed open=false Stacks remain discoverable | `TestCompletedStackRemainsDecodableAndDiscoverable` |
| 19 | Safe handling of server-side SHA/base changes | `TestForcePushWithLeaseUsesAtomicExplicitExpectations`, `TestValidateWritePlanChecksChainRepositoryAndLifecycle` |
| 20 | Unsupported structural mutations are refused | conflict cases in `TestClassify`, `TestNativeLandRefusalHasNoMutationPath` |
