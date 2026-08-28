/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	wave1PlanningGrantPath            = ".harness/grants/WAVE-1-contract-publication.yaml"
	wave1PlanningGrantSignature       = ".harness/grants/WAVE-1-contract-publication.yaml.sig"
	wave1PlanningGrantKey             = ".harness/keys/genesis-signing-key.pub"
	wave1PlanningGrantNamespace       = "mars3-planning-grant"
	wave1PlanningGrantBase            = "ee385ce236ae1f99da692d223d7666b80dd9108f"
	wave1PlanningGrantBranch          = "codex/w-001-work-authority"
	planningGrantCommitNS             = "git"
	planningGrantRepository           = "greaveselliott/MARS-3"
	planningGrantWorkflow             = "Foundation quality"
	planningGrantWorkflowPath         = ".github/workflows/foundation-quality.yml"
	planningGrantWorkflowJob          = "public-commit-gate"
	wave1DispositionPath              = ".harness/grants/WAVE-1-recovery-disposition.yaml"
	wave1DispositionSignature         = ".harness/grants/WAVE-1-recovery-disposition.yaml.sig"
	wave1DispositionSnapshot          = ".harness/grants/WAVE-1-authority-recovery-state.json"
	wave1DispositionNamespace         = "mars3-recovery-disposition"
	wave1DispositionSnapshotSHA       = "4d3b5c9d90a223c0e9d974e836559309a2f4dac7f209a3966336e9152f57feca"
	wave1PriorPublicationTag          = "mars3/wave1-contract-publication-v1"
	wave1PriorPublicationTagMessage   = "MARS-3 Wave-1 contract-publication tree attestation v1"
	wave1PriorPublicationTagObject    = "4bce7e7d4a8b2cc1a5b30b9feaee61232c3cc0de"
	wave1AddendumPath                 = ".harness/grants/WAVE-1-ci-recovery-addendum.yaml"
	wave1AddendumSignature            = ".harness/grants/WAVE-1-ci-recovery-addendum.yaml.sig"
	wave1AddendumNamespace            = "mars3-ci-recovery-addendum"
	wave1AddendumBase                 = "a22cfe6fada6f2bc787742eae50bca28cec80c89"
	wave1AddendumBaseTree             = "3c5befaefab37a8d0a2e3a8af2efd6e1eb1d8cae"
	wave1V2PublicationTag             = "mars3/wave1-contract-publication-v2"
	wave1V2PublicationTagMessage      = "MARS-3 Wave-1 contract-publication tree attestation v2"
	wave1V2PublicationTagObject       = "e334356519188fc0906549515ae57fbffa646829"
	wave1V3AddendumPath               = ".harness/grants/WAVE-1-ci-recovery-addendum-v3.yaml"
	wave1V3AddendumSignature          = ".harness/grants/WAVE-1-ci-recovery-addendum-v3.yaml.sig"
	wave1V3AddendumNamespace          = "mars3-ci-recovery-addendum-v3"
	wave1V3AddendumBase               = "412a9b857265af250ee40d36d0a6c127714e4ec9"
	wave1V3AddendumBaseTree           = "8c7f3ccac3e31d0e8b45431934cd95a91e448c0f"
	wave1V3ObservedStaleMerge         = "fff2bea9bffa9400d3ecfc147b7338849ecfbbb0"
	wave1PublicationTag               = "mars3/wave1-contract-publication-v3"
	wave1PublicationTagMessage        = "MARS-3 Wave-1 contract-publication tree attestation v3"
	wave1PublishedMain                = "265b0b78a19d0ac50611c360f4614ed24d1cfcd7"
	wave1PublishedTree                = "87dc32acce9c767f01f94aae25936665e26650ab"
	wave1DirectMainGrantPath          = ".harness/grants/WAVE-1-direct-main-transition.yaml"
	wave1DirectMainGrantSignature     = ".harness/grants/WAVE-1-direct-main-transition.yaml.sig"
	wave1DirectMainGrantNamespace     = "mars3-direct-main-transition"
	wave1PRFallbackPath               = ".harness/grants/WAVE-1-pr-fallback-addendum.yaml"
	wave1PRFallbackSignature          = ".harness/grants/WAVE-1-pr-fallback-addendum.yaml.sig"
	wave1PRFallbackNamespace          = "mars3-pr-fallback-addendum"
	wave1PRFallbackBranch             = "codex/w-001-bootstrap-transition"
	wave1PRFallbackFirstCommit        = "0602a7e2036f53d2d7b9da10b406d27f87621196"
	wave1PRFallbackFirstTree          = "69d9ab152f5b0a243855ddf613192c662b52306b"
	wave1TransitionTag                = "mars3/w001-transition-v1"
	wave1TransitionTagMessage         = "MARS-3 W-001 transition tree attestation v1"
	wave1TransitionTagObject          = "394c9ce749142c2222c1b8081b62f43a895be326"
	wave1TransitionReviewedHead       = "a9e203540a9e280fe7059dfa579a9a4089fc3f52"
	wave1TransitionReviewedTree       = "c326c5d33f075a2c8b7dc4c41a698d57175c8805"
	wave1MainCIFixPath                = ".harness/grants/WAVE-1-pr-fallback-main-ci-addendum.yaml"
	wave1MainCIFixSignature           = ".harness/grants/WAVE-1-pr-fallback-main-ci-addendum.yaml.sig"
	wave1MainCIFixNamespace           = "mars3-pr-fallback-main-ci-addendum"
	wave1SuccessorTransitionTag       = "mars3/w001-transition-v2"
	wave1SuccessorTagMessage          = "MARS-3 W-001 transition tree attestation v2"
	wave1SuccessorTagObject           = "cd7d6daeea77041167b5aa3763952b47b4ad09c0"
	wave1CIFixtureReviewedHead        = "b9ca75dfe42001a9632d8c752e2ddb80624fa4ae"
	wave1CIFixtureReviewedTree        = "f8af794a23f67fec73af71328dc6aeae0fa9e104"
	wave1CIFixtureFixPath             = ".harness/grants/WAVE-1-pr-fallback-fixture-stabilization-addendum.yaml"
	wave1CIFixtureFixSignature        = ".harness/grants/WAVE-1-pr-fallback-fixture-stabilization-addendum.yaml.sig"
	wave1CIFixtureFixNamespace        = "mars3-pr-fallback-fixture-stabilization-addendum"
	wave1FinalTransitionTag           = "mars3/w001-transition-v3"
	wave1FinalTransitionTagMessage    = "MARS-3 W-001 transition tree attestation v3"
	w001BootstrapGrantPath            = ".harness/grants/W-001-bootstrap.yaml"
	w001BootstrapGrantSignature       = ".harness/grants/W-001-bootstrap.yaml.sig"
	w001BootstrapGrantNamespace       = "mars3-w001-bootstrap-grant"
	w001BootstrapExecutionNamespace   = "mars3-w001-bootstrap-execution"
	w001BootstrapBase                 = "37b55b912b20715349bc50e0524c85d4b22f1772"
	w001BootstrapBaseTree             = "f06864b0802cea793cf7a0c08b60b7e734539a94"
	w001BootstrapBranch               = "codex/w-001-work-authority"
	w001BootstrapReviewTag            = "mars3/w001-bootstrap-helper-v9"
	w001BootstrapReviewTagMessage     = "MARS-3 W-001 bootstrap helper tree attestation v9"
	w001PostclaimGrantPath            = ".harness/grants/W-001-postclaim-reconciliation.yaml"
	w001PostclaimGrantSignature       = ".harness/grants/W-001-postclaim-reconciliation.yaml.sig"
	w001PostclaimGrantNamespace       = "mars3-w001-postclaim-reconciliation"
	w001PostclaimBase                 = "adfd64feb565fb703a3568122cc032d4d1a450f5"
	w001PostclaimBaseTree             = "bbfa7f59f7bd29c1a0546ffcb77dd8fa4982ef6d"
	w001PostclaimBranch               = "codex/w-001-postclaim-reconciliation"
	w001PostclaimReviewTag            = "mars3/w001-postclaim-reconciliation-v1"
	w001PostclaimReviewTagMessage     = "MARS-3 W-001 postclaim reconciliation tree attestation v1"
	w001PostclaimCIFixPath            = ".harness/grants/W-001-postclaim-ci-stabilization-v2.yaml"
	w001PostclaimCIFixSignature       = ".harness/grants/W-001-postclaim-ci-stabilization-v2.yaml.sig"
	w001PostclaimCIFixNamespace       = "mars3-w001-postclaim-ci-stabilization-v2"
	w001PostclaimCIFixBase            = "eda666569de379543a170119ccb7c560478c7346"
	w001PostclaimCIFixBaseTree        = "b8de9ac4cf26b4561ce26abcf729529f65bd9b9f"
	w001PostclaimCIFixReviewTag       = "mars3/w001-postclaim-reconciliation-v2"
	w001PostclaimCIFixTagMessage      = "MARS-3 W-001 postclaim reconciliation tree attestation v2"
	w001PostclaimV1TagObject          = "7492f63fd88567a284eff43c670098295824aaf8"
	w001PostclaimV2TagObject          = "684ba6c536e8e8d4c33b587da9e7d05893168958"
	w001PostclaimSecurityFixPath      = ".harness/grants/W-001-postclaim-security-correction-v3.yaml"
	w001PostclaimSecurityFixSig       = ".harness/grants/W-001-postclaim-security-correction-v3.yaml.sig"
	w001PostclaimSecurityFixNS        = "mars3-w001-postclaim-security-correction-v3"
	w001PostclaimSecurityFixBase      = "20542d8e696abe0a71b6ec3ceb23f042919fbc04"
	w001PostclaimSecurityFixTree      = "499ea91f5002b32d57d6f20b4ca3ea07dbdc73f5"
	w001PostclaimSecurityFixTag       = "mars3/w001-postclaim-reconciliation-v3"
	w001PostclaimSecurityFixTagMsg    = "MARS-3 W-001 postclaim reconciliation tree attestation v3"
	w001PostclaimSecurityHelperSHA    = "21ff880af1135db6c13ad6dcfac621e9231fee8e635ca7d47e5fcc5c39314f25"
	w001PostclaimSecurityTestSHA      = "0221972a39c14108719b80a23448f9c42f2b9417a3d4ea3dc4f5c1b2975a0f33"
	w001PostclaimSecurityBasePatchSHA = "50128252828352366ced6560371468a5746c2603ef89ea746a33be8994ffceb6"
	w001PostclaimSecurityPatchPath    = "internal/authority/bootstrap/beads-v1.2.2-effective-db-security.patch"
	w001PostclaimSecurityPatchSHA     = "d48b398a8688d337192ab030c69fd9df0809f72051da7850ff2fdbad5e322d45"
	w001PostclaimSecurityBinarySHA    = "8d13927671519fd74470820a72c6ff069589655e338649f31db7e654b2b36c00"
	w001PostclaimHookFixPath          = ".harness/grants/W-001-postclaim-hook-isolation-v4.yaml"
	w001PostclaimHookFixSig           = ".harness/grants/W-001-postclaim-hook-isolation-v4.yaml.sig"
	w001PostclaimHookFixNS            = "mars3-w001-postclaim-hook-isolation-v4"
	w001PostclaimHookFixBase          = "9e8a587f8187c2d385a6c5fa023346405733d7ff"
	w001PostclaimHookFixTree          = "fb4a0abea8f3e77a92ca86672078d2d68df7a9e0"
	w001PostclaimHookFixTag           = "mars3/w001-postclaim-reconciliation-v4"
	w001PostclaimHookFixTagMsg        = "MARS-3 W-001 postclaim reconciliation tree attestation v4"
	w001PostclaimV3TagObject          = "004fde1244312e088a4397809cb7ab3a81706612"
	w001PostclaimHookHelperSHA        = "cc8e102d8e4aa26e847f14a6a39522b20f193b6419b147678a01db3eb6bcab21"
	w001PostclaimHookTestSHA          = "a1be6df7bfddb10ce964471af484b71ecb4399ab840c8062d28696a87662e809"
	w001PostclaimHookPatchPath        = "internal/authority/bootstrap/beads-v1.2.2-bootstrap-hook-isolation.patch"
	w001PostclaimHookPatchSHA         = "fc282ebc257fc41c15ab7b8ffdd50a1600f840254cf6e10b6997f9e30a0dc1fc"
	w001PostclaimHookBinarySHA        = "22042fc0844ab7700417d917c386f2eab4bab5dd6a6404be091cbd5edbe9e154"
	w001PostclaimPRFixPath            = ".harness/grants/W-001-postclaim-pr-binding-v5.yaml"
	w001PostclaimPRFixSig             = ".harness/grants/W-001-postclaim-pr-binding-v5.yaml.sig"
	w001PostclaimPRFixNS              = "mars3-w001-postclaim-pr-binding-v5"
	w001PostclaimPRFixBase            = "d890a96014f79438d36bde3c8967664163e9d961"
	w001PostclaimPRFixTree            = "3d3252b0559e664203521af2e85d0d87cdb9fcd1"
	w001PostclaimPRFixTag             = "mars3/w001-postclaim-reconciliation-v5"
	w001PostclaimPRFixTagMsg          = "MARS-3 W-001 postclaim reconciliation tree attestation v5"
	w001PostclaimV4TagObject          = "d50d55bfe49d159979dbee26122319978a7ae612"
	w001PostclaimActivePR             = 8
	w001PostclaimChronoFixPath        = ".harness/grants/W-001-postclaim-chronology-correction-v6.yaml"
	w001PostclaimChronoFixSig         = ".harness/grants/W-001-postclaim-chronology-correction-v6.yaml.sig"
	w001PostclaimChronoFixNS          = "mars3-w001-postclaim-chronology-correction-v6"
	w001PostclaimChronoFixBase        = "765a07d3ebe2432227de7ccad65dc9f3b291deba"
	w001PostclaimChronoFixTree        = "ce8959ecd2e99c5181651fa9f2eca926da971e47"
	w001PostclaimChronoFixTag         = "mars3/w001-postclaim-reconciliation-v6"
	w001PostclaimChronoFixTagMsg      = "MARS-3 W-001 postclaim reconciliation tree attestation v6"
	w001PostclaimV5TagObject          = "31f7de15ef790d795f80de32ca0cf459c192cc5e"
	w001PostclaimV6TagObject          = "2346a1388272569bb64817ea7e9b6463c4e84e5a"
	w001DeliveryGrantPath             = ".harness/grants/W-001-delivery.yaml"
	w001DeliveryGrantSignature        = ".harness/grants/W-001-delivery.yaml.sig"
	w001DeliveryGrantNamespace        = "mars3-w001-delivery"
	w001DeliveryBase                  = "59f1fe24952b68bd3bbb6994bfee46c350b7c9cd"
	w001DeliveryBaseTree              = "7febda7ec2fec47b7d6bf11fdd5b24e605b9e2b2"
	w001DeliveryBranch                = "codex/w-001-delivery-v2"
	w001DeliveryReviewTag             = "mars3/w001-delivery-v2"
	w001DeliveryReviewTagMessage      = "MARS-3 W-001 delivery tree attestation v2"
	w001DeliveryCIFixPath             = ".harness/grants/W-001-delivery-ci-correction-v3.yaml"
	w001DeliveryCIFixSignature        = ".harness/grants/W-001-delivery-ci-correction-v3.yaml.sig"
	w001DeliveryCIFixNamespace        = "mars3-w001-delivery-ci-correction-v3"
	w001DeliveryCIFixBase             = "ac20b235724b2219e5db230a7a44b507e46d5547"
	w001DeliveryCIFixBaseTree         = "4812b71b88500101688be7c80f41461a79619646"
	w001DeliveryV2TagObject           = "9eb770c85a1df06dd90e993c9447176c9bbbffd0"
	w001DeliveryCIFixReviewTag        = "mars3/w001-delivery-v3"
	w001DeliveryCIFixReviewTagMessage = "MARS-3 W-001 delivery tree attestation v3"
	w001DeliveryScannerFixPath        = ".harness/grants/W-001-delivery-scanner-correction-v4.yaml"
	w001DeliveryScannerFixSignature   = ".harness/grants/W-001-delivery-scanner-correction-v4.yaml.sig"
	w001DeliveryScannerFixNamespace   = "mars3-w001-delivery-scanner-correction-v4"
	w001DeliveryScannerFixBase        = "383ea617ad2bcbe06522a30014a1b19127b5239f"
	w001DeliveryScannerFixBaseTree    = "e91776b8de9e9d1e1e193ae9588363c4d87a62e6"
	w001DeliveryV3TagObject           = "700d85715981fb6e9def191b414c815c8f543dd0"
	w001DeliveryScannerFixReviewTag   = "mars3/w001-delivery-v4"
	w001DeliveryScannerFixTagMessage  = "MARS-3 W-001 delivery tree attestation v4"
	w001DeliveryV1PreservedHead       = "919f1189fb0703e42bcc11570a59527ad8e7a444"
	w001DeliveryScannerIgnorePath     = ".gitleaksignore"
	w001LifecycleGrantPath            = ".harness/grants/W-001-lifecycle-completion-v5.yaml"
	w001LifecycleGrantSignature       = ".harness/grants/W-001-lifecycle-completion-v5.yaml.sig"
	w001LifecycleGrantNamespace       = "mars3-w001-lifecycle-completion-v5"
	w001LifecycleBase                 = "7f35c8a7112946a9569efe6085f49da8fd28530e"
	w001LifecycleBaseTree             = "5a9f006b0cd65364c2fdcfb403efd554f0e34dda"
	w001LifecycleBranch               = "codex/w-001-lifecycle-completion"
	w001LifecycleReviewTag            = "mars3/w001-lifecycle-completion-v5"
	w001LifecycleReviewTagMessage     = "MARS-3 W-001 lifecycle completion tree attestation v5"
	w001DeliveryV4TagObject           = "98a3f34c24868e49ca4909c8b0303f34c25390f3"
	w001LifecycleCorrectionPath       = ".harness/grants/W-001-lifecycle-correction-v6.yaml"
	w001LifecycleCorrectionSignature  = ".harness/grants/W-001-lifecycle-correction-v6.yaml.sig"
	w001LifecycleCorrectionNamespace  = "mars3-w001-lifecycle-correction-v6"
	w001LifecycleCorrectionBase       = "523ead6f899c413cb0a388c60a30b33aed88b8b6"
	w001LifecycleCorrectionBaseTree   = "aaff531b1b0fee9dfa907a5a52c0afd98abf050c"
	w001LifecycleV5TagObject          = "15dbd1be9d1d098eb2f5da3dbafe824064dbff1f"
	w001LifecycleCorrectionReviewTag  = "mars3/w001-lifecycle-completion-v6"
	w001LifecycleCorrectionTagMessage = "MARS-3 W-001 lifecycle correction tree attestation v6"
)

const (
	w001LifecycleCorrectionV7Path           = ".harness/grants/W-001-lifecycle-correction-v7.yaml"
	w001LifecycleCorrectionV7Signature      = ".harness/grants/W-001-lifecycle-correction-v7.yaml.sig"
	w001LifecycleCorrectionV7Namespace      = "mars3-w001-lifecycle-correction-v7"
	w001LifecycleCorrectionV7Base           = "e0f27046ec28ab924eac910d40e244cb26b30323"
	w001LifecycleCorrectionV7BaseTree       = "bbc1aa76b3965f0740e54f984ad713978a3be9f8"
	w001LifecycleV6TagObject                = "d8637c7443ab04e05892ecf5489f0b45fa41e43d"
	w001LifecycleCorrectionV7ReviewTag      = "mars3/w001-lifecycle-completion-v7"
	w001LifecycleCorrectionV7TagMessage     = "MARS-3 W-001 lifecycle correction tree attestation v7"
	w001LifecycleCorrectionV7PatchPath      = "internal/authority/beads/beads-v1.2.2-lifecycle.patch"
	w001LifecycleCorrectionV7PatchSHA       = "2db1615df7bc1c5b4bd0d2d17cecb22a43b2bf4be72a1ebcf750820170b5ff66"
	w001LifecycleCorrectionV8Path           = ".harness/grants/W-001-lifecycle-correction-v8.yaml"
	w001LifecycleCorrectionV8Signature      = ".harness/grants/W-001-lifecycle-correction-v8.yaml.sig"
	w001LifecycleCorrectionV8Namespace      = "mars3-w001-lifecycle-correction-v8"
	w001LifecycleCorrectionV8Base           = "36d8c981ebde65e694416caf16fc02d50aac2a67"
	w001LifecycleCorrectionV8BaseTree       = "be55454779c2c0dd08adc08666c2b7ee3826448f"
	w001LifecycleV7TagObject                = "217585d45ec414f55e5d326419a4f79b96a48915"
	w001LifecycleCorrectionV8ReviewTag      = "mars3/w001-lifecycle-completion-v8"
	w001LifecycleCorrectionV8TagMessage     = "MARS-3 W-001 lifecycle correction tree attestation v8"
	w001LifecycleCorrectionV8PatchSHA       = "116c3b59744f1d6c3065ef8baf89d2bfac372bab66282b8cd9443e0843fc65c5"
	w001LifecycleCorrectionV9Path           = ".harness/grants/W-001-lifecycle-correction-v9.yaml"
	w001LifecycleCorrectionV9Signature      = ".harness/grants/W-001-lifecycle-correction-v9.yaml.sig"
	w001LifecycleCorrectionV9Namespace      = "mars3-w001-lifecycle-correction-v9"
	w001LifecycleCorrectionV9Base           = "6d6b90ef495cd64286e755e90d199a3cb622cd54"
	w001LifecycleCorrectionV9BaseTree       = "f596e2a148f055bcac90960419b2e22928bd471c"
	w001LifecycleV8TagObject                = "fb99ef24abb1176e7bcec01bddffe305979d8464"
	w001LifecycleCorrectionV9ReviewTag      = "mars3/w001-lifecycle-completion-v9"
	w001LifecycleCorrectionV9TagMessage     = "MARS-3 W-001 lifecycle correction tree attestation v9"
	w001LifecycleCorrectionV9PatchSHA       = "6cca8ab8bd5bd0d5f179612ece7e68e002caa69c455c80cdb00335d5e75a31c4"
	w001LifecycleStabilizationV10Path       = ".harness/grants/W-001-lifecycle-ci-stabilization-v10.yaml"
	w001LifecycleStabilizationV10Signature  = ".harness/grants/W-001-lifecycle-ci-stabilization-v10.yaml.sig"
	w001LifecycleStabilizationV10Namespace  = "mars3-w001-lifecycle-ci-stabilization-v10"
	w001LifecycleStabilizationV10Base       = "ad845ff81f1e64b9e4110162a77a65a844891731"
	w001LifecycleStabilizationV10BaseTree   = "e4a08e5a4b211003dc29609a0128856eec306061"
	w001LifecycleV9TagObject                = "47933c4957b9af2e8d7a38f971d7a20c5de8122f"
	w001LifecycleStabilizationV10ReviewTag  = "mars3/w001-lifecycle-completion-v10"
	w001LifecycleStabilizationV10TagMessage = "MARS-3 W-001 lifecycle CI stabilization tree attestation v10"
	w001LifecycleCIFencingV11Path           = ".harness/grants/W-001-lifecycle-ci-fencing-v11.yaml"
	w001LifecycleCIFencingV11Signature      = ".harness/grants/W-001-lifecycle-ci-fencing-v11.yaml.sig"
	w001LifecycleCIFencingV11Namespace      = "mars3-w001-lifecycle-ci-fencing-v11"
	w001LifecycleCIFencingV11Base           = "47b19b2c89d72fbf9eb5356ceefe33783d691aa4"
	w001LifecycleCIFencingV11BaseTree       = "0ebe496c48871b040a7fcd7a286073f2c1d40153"
	w001LifecycleV10TagObject               = "84672df5f046995bb7efd79cf8f9a333946aecfa"
	w001LifecycleCIFencingV11ReviewTag      = "mars3/w001-lifecycle-completion-v11"
	w001LifecycleCIFencingV11TagMessage     = "MARS-3 W-001 lifecycle CI Git fencing correction tree attestation v11"
	w001LifecycleCIHardeningV12Path         = ".harness/grants/W-001-lifecycle-ci-hardening-v12.yaml"
	w001LifecycleCIHardeningV12Signature    = ".harness/grants/W-001-lifecycle-ci-hardening-v12.yaml.sig"
	w001LifecycleCIHardeningV12Namespace    = "mars3-w001-lifecycle-ci-hardening-v12"
	w001LifecycleCIHardeningV12Base         = "54f4593b1730ff9ae04a2e5cce0589c6baedfee6"
	w001LifecycleCIHardeningV12BaseTree     = "44ba564be30e0db0aa735d76539c3604a5d79e3f"
	w001LifecycleV11TagObject               = "7313ee2e38dd1d4f4f5ca62237e0be89b0b4f13a"
	w001LifecycleCIHardeningV12ReviewTag    = "mars3/w001-lifecycle-completion-v12"
	w001LifecycleCIHardeningV12TagMessage   = "MARS-3 W-001 lifecycle CI hardening tree attestation v12"
	w001LifecycleCIHardeningV13Path         = ".harness/grants/W-001-lifecycle-ci-hardening-v13.yaml"
	w001LifecycleCIHardeningV13Signature    = ".harness/grants/W-001-lifecycle-ci-hardening-v13.yaml.sig"
	w001LifecycleCIHardeningV13Namespace    = "mars3-w001-lifecycle-ci-hardening-v13"
	w001LifecycleCIHardeningV13Base         = "3c8d55aa39e4e099d8a922f8e13a71efcbe2c78b"
	w001LifecycleCIHardeningV13BaseTree     = "c4bb80ab477b7fcbe73a7a237479e44703393952"
	w001LifecycleV12TagObject               = "d0176029978e0c49d795a02ad36f7f7992c3bdfa"
	w001LifecycleCIHardeningV13ReviewTag    = "mars3/w001-lifecycle-completion-v13"
	w001LifecycleCIHardeningV13TagMessage   = "MARS-3 W-001 lifecycle CI closed argv and process admission tree attestation v13"
	w001LifecycleCIHardeningV14Path         = ".harness/grants/W-001-lifecycle-ci-hardening-v14.yaml"
	w001LifecycleCIHardeningV14Signature    = ".harness/grants/W-001-lifecycle-ci-hardening-v14.yaml.sig"
	w001LifecycleCIHardeningV14Namespace    = "mars3-w001-lifecycle-ci-hardening-v14"
	w001LifecycleCIHardeningV14Base         = "ce934054aed66c074e99a032191a6a51c620b947"
	w001LifecycleCIHardeningV14BaseTree     = "73cab7fb7b1bd2fc1102dc4b16e9617fd7c26680"
	w001LifecycleV13TagObject               = "8d50fbb230503f4ad24cfc7301e4e4924be30ec0"
	w001LifecycleCIHardeningV14ReviewTag    = "mars3/w001-lifecycle-completion-v14"
	w001LifecycleCIHardeningV14TagMessage   = "MARS-3 W-001 lifecycle CI physical path and transitive process boundary tree attestation v14"
	w001LifecycleCIHardeningV15Path         = ".harness/grants/W-001-lifecycle-ci-hardening-v15.yaml"
	w001LifecycleCIHardeningV15Signature    = ".harness/grants/W-001-lifecycle-ci-hardening-v15.yaml.sig"
	w001LifecycleCIHardeningV15Namespace    = "mars3-w001-lifecycle-ci-hardening-v15"
	w001LifecycleCIHardeningV15Base         = "d631bec4ed786116c13e36995722d91d48d64109"
	w001LifecycleCIHardeningV15BaseTree     = "b9467f12b2031c5159ef749938bbd4f475eb6153"
	w001LifecycleV14TagObject               = "f97b9ebd0150ee4d75bf691c16c176b792d42461"
	w001LifecycleCIHardeningV15ReviewTag    = "mars3/w001-lifecycle-completion-v15"
	w001LifecycleCIHardeningV15TagMessage   = "MARS-3 W-001 lifecycle CI descriptor-bound Git and closed process import tree attestation v15"
	w001LifecycleCIHardeningV16Path         = ".harness/grants/W-001-lifecycle-ci-hardening-v16.yaml"
	w001LifecycleCIHardeningV16Signature    = ".harness/grants/W-001-lifecycle-ci-hardening-v16.yaml.sig"
	w001LifecycleCIHardeningV16Namespace    = "mars3-w001-lifecycle-ci-hardening-v16"
	w001LifecycleCIHardeningV16Base         = "a46f16deff2fc06c5d0d21377a3bb2c65e873fc9"
	w001LifecycleCIHardeningV16BaseTree     = "c2e482717f182040708cbf2551ee266de2485a30"
	w001LifecycleV15TagObject               = "71230aed1661a987dbd1b63b058180a6b33f7825"
	w001LifecycleCIHardeningV16ReviewTag    = "mars3/w001-lifecycle-completion-v16"
	w001LifecycleCIHardeningV16TagMessage   = "MARS-3 W-001 lifecycle CI closed descriptor launcher attestation v16"
	w001LifecycleCIHardeningV17Path         = ".harness/grants/W-001-lifecycle-ci-hardening-v17.yaml"
	w001LifecycleCIHardeningV17Signature    = ".harness/grants/W-001-lifecycle-ci-hardening-v17.yaml.sig"
	w001LifecycleCIHardeningV17Namespace    = "mars3-w001-lifecycle-ci-hardening-v17"
	w001LifecycleCIHardeningV17Base         = "25d2f14e20e74f1415caa4118a93c359f9370031"
	w001LifecycleCIHardeningV17BaseTree     = "d9bf0e3f89807c12c5be5a58ea68fd04715aa740"
	w001LifecycleV16TagObject               = "125a596c00a5a00f40fbda002f38cc06e3f0b5cb"
	w001LifecycleCIHardeningV17ReviewTag    = "mars3/w001-lifecycle-completion-v17"
	w001LifecycleCIHardeningV17TagMessage   = "MARS-3 W-001 lifecycle CI descriptor-stream and closed executor attestation v17"
)

// W001BootstrapGrant is the validated public projection consumed by the
// one-shot bootstrap helper. It contains no local path, credential, or signer
// state.
type W001BootstrapGrant struct {
	ID                          string
	AttemptID                   string
	IdempotencyKey              string
	Bead                        string
	BaseCommit                  string
	WorkingBranch               string
	Assignee                    string
	AuthorityProjectID          string
	ExpectedNativeStatus        string
	ExpectedLifecycleState      string
	ExpectedCreatedAt           string
	ExpectedUpdatedAt           string
	ExpectedMetadataSHA256      string
	ExpectedLabelsSHA256        string
	ExpectedDependency          string
	ExpectedDependencyType      string
	ExpectedDependencyStatus    string
	ExpectedDependencyLifecycle string
	ExpectedDependencySHA256    string
	ExpectedLineageSHA256       string
	ExpiresAt                   string
	PostNativeStatus            string
	PostLifecycleState          string
	PostMetadataSHA256          string
	PostLabelsSHA256            string
	PostMetadataBase64          string
	RemoveLabel                 string
	AddLabel                    string
	BeadsVersion                string
	BeadsSourceCommit           string
	BeadsBinarySHA256           string
	DoltModule                  string
	DoltModuleSHA256            string
	GoVersion                   string
	GoOS                        string
	GoArch                      string
	ICUFormula                  string
	DoltTestImage               string
	PatchPath                   string
	PatchSHA256                 string
	CorrectionPatchPath         string
	CorrectionPatchSHA256       string
	HookIsolationPatchPath      string
	HookIsolationPatchSHA256    string
	PatchedBinarySHA256         string
	GoBinarySHA256              string
	ReviewTag                   string
}

// W001BootstrapExecutionAuthorization is a short-lived, independently signed
// post-review effect token. It is intentionally external to Git so it can bind
// the actual protected-main commit and check after the helper is merged.
type W001BootstrapExecutionAuthorization struct {
	SchemaVersion           int    `json:"schemaVersion"`
	Kind                    string `json:"kind"`
	Classification          string `json:"classification"`
	GrantID                 string `json:"grantId"`
	Repository              string `json:"repository"`
	AttemptID               string `json:"attemptId"`
	IdempotencyKey          string `json:"idempotencyKey"`
	Bead                    string `json:"bead"`
	AuthorityProjectID      string `json:"authorityProjectId"`
	MergedCommit            string `json:"mergedCommit"`
	MergedTree              string `json:"mergedTree"`
	ReviewTag               string `json:"reviewTag"`
	ReviewedFeatureCommit   string `json:"reviewedFeatureCommit"`
	PullRequest             int    `json:"pullRequest"`
	ProtectedMainCheckRun   int64  `json:"protectedMainCheckRun"`
	QAReviewedCommit        string `json:"qaReviewedCommit"`
	QADisposition           string `json:"qaDisposition"`
	SecurityReviewedCommit  string `json:"securityReviewedCommit"`
	SecurityDisposition     string `json:"securityDisposition"`
	PatchedBinarySHA256     string `json:"patchedBinarySHA256"`
	ExpectedMetadataSHA256  string `json:"expectedMetadataSHA256"`
	WorkspaceInstanceSHA256 string `json:"workspaceInstanceSHA256"`
	AllowedEffect           string `json:"allowedEffect"`
	IssuedAt                string `json:"issuedAt"`
	ExpiresAt               string `json:"expiresAt"`
	payloadSHA256           string
	signatureSHA256         string
}

// W001DeliveryGrant is the validated runtime projection of the signed W-001
// delivery authority. It deliberately excludes signing material and exposes
// only the fields required to reconcile the already-canonical claim with one
// bounded development lease.
type W001DeliveryGrant struct {
	ID                           string
	Repository                   string
	Bead                         string
	Principal                    string
	AttemptID                    string
	IdempotencyKey               string
	BaseCommit                   string
	ExpiresAt                    time.Time
	ExpectedNativeStatus         string
	ExpectedLifecycleState       string
	ExpectedAssignee             string
	CanonicalClaimAttemptID      string
	WorkVersionGeneration        string
	WorkVersionIncarnation       string
	IssueMutationSequence        uint64
	DependencyGraphRevision      uint64
	CanonicalWorkMutationAllowed bool
	DevelopmentLeaseAllowed      bool
}

type grantScalarExpectation struct {
	path  string
	value string
}

var wave1PlanningGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3ContractPublicationGrant"},
	{path: "grant.id", value: "WAVE-1-contract-publication"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T05:30:55Z"},
	{path: "grant.baseCommit", value: wave1PlanningGrantBase},
	{path: "grant.workingBranch", value: wave1PlanningGrantBranch},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.purpose", value: "publish reviewed Wave-1 Git contracts and admission validators before any implementation claim"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.implementationClaimed", value: "false"},
	{path: "grant.expiresOn", value: "first verified merge of this grant and its bounded contract-publication change to main"},
	{path: "grant.successorAuthority", value: "signed W-001 implementation bootstrap grant or the accepted authority gateway"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.rawAuthorityPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1PlanningGrantNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-contract-publication.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1DirectMainGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3DirectMainTransitionGrant"},
	{path: "grant.id", value: "WAVE-1-direct-main-transition"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T08:30:17Z"},
	{path: "grant.expiresAt", value: "2026-08-29T08:30:17Z"},
	{path: "grant.baseCommit", value: wave1PublishedMain},
	{path: "grant.baseTree", value: wave1PublishedTree},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.workingBranch", value: "main"},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.sourceDirective", value: "commit-small-semantic-changes-directly-to-main-frequently"},
	{path: "grant.purpose", value: "prepare fail-closed admission for the separately signed W-001 bootstrap grant without claiming or implementing W-001"},
	{path: "grant.priorPublicationTag", value: wave1PublicationTag},
	{path: "grant.priorPublicationTagObject", value: "b53728e3a57e6dc0d57151aa7f0bed8e44aaaa2f"},
	{path: "grant.priorPublicationTagTarget", value: "7e6b765c284788442553d40792db0afb128c4872"},
	{path: "grant.priorPublicationTree", value: wave1PublishedTree},
	{path: "grant.successorGrant", value: "W-001-bootstrap"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.implementationClaimed", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "grant.githubPolicyMutationAllowed", value: "false"},
	{path: "grant.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequiredAfterEveryCommit", value: "true"},
	{path: "verification.directMainPushRequired", value: "true"},
	{path: "verification.exactBaseAndTreeRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1DirectMainGrantNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-direct-main-transition.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1DirectMainGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"edit-listed-public-paths",
		"create-pinned-signer-direct-main-commits",
		"push-protected-main",
		"run-public-commit-gate-after-every-commit",
		"prepare-separately-signed-W001-bootstrap-grant",
		"append-public-safe-transition-intent-and-receipt",
	},
	"grant.authorizedPaths": {
		wave1DirectMainGrantPath,
		wave1DirectMainGrantSignature,
		".harness/grants/W-001-bootstrap.yaml",
		".harness/grants/W-001-bootstrap.yaml.sig",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"every-post-publication-commit-is-linear-and-pinned-signer-verified",
		"every-protected-main-push-binds-event-before-after-and-first-parent",
		"every-commit-and-worktree-path-is-inside-the-active-signed-authority",
		"contract-publication-remains-backlog-unclaimed-until-the-successor-grant-and-claim-receipt-validate",
		"v1-v2-v3-publication-tags-remain-immutable",
		"no-transition-gap-is-selected-by-mutable-plan-text",
	},
	"grant.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"move-or-delete-v1-v2-or-v3-publication-tag",
		"create-or-move-another-publication-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1PRFallbackScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3PRFallbackAddendum"},
	{path: "addendum.id", value: "WAVE-1-pr-fallback-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T08:40:31Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T08:40:31Z"},
	{path: "addendum.baseCommit", value: wave1PublishedMain},
	{path: "addendum.baseTree", value: wave1PublishedTree},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.workingBranch", value: wave1PRFallbackBranch},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.sourceDirective", value: "if-using-prs-validate-and-merge-them-promptly"},
	{path: "addendum.purpose", value: "route the signed W-001 transition gate through the repository-required pull-request check without weakening protected main"},
	{path: "addendum.priorGrant", value: "WAVE-1-direct-main-transition"},
	{path: "addendum.priorGrantSHA256", value: "0e8a8179c8fb44d6a23421955bdfb74878cc7cb21cbb32b8b589a2fa9e99e4ba"},
	{path: "addendum.priorGrantSignatureSHA256", value: "93ab5b024bc859eea7d5a3adc026cfd87636c6160cfee07b99fc8f8e8f4d3555"},
	{path: "addendum.rejectedDirectCommit", value: wave1PRFallbackFirstCommit},
	{path: "addendum.rejectedDirectTree", value: wave1PRFallbackFirstTree},
	{path: "addendum.rejectedDirectParent", value: wave1PublishedMain},
	{path: "addendum.failureFingerprint", value: "github-rules-direct-main-requires-pr"},
	{path: "addendum.rulesetId", value: "21510926"},
	{path: "addendum.retryDirectPush", value: "false"},
	{path: "addendum.mergeMode", value: "squash"},
	{path: "addendum.transitionTag", value: wave1TransitionTag},
	{path: "addendum.transitionTagMessage", value: wave1TransitionTagMessage},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.protectedMainUnchangedUntilMerge", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1PRFallbackNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-pr-fallback-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1PRFallbackSequences = map[string][]string{
	"addendum.allowedEffects": {
		"create-exact-working-branch-at-rejected-direct-commit",
		"edit-listed-public-paths",
		"create-pinned-signer-correction-commit",
		"create-and-push-one-pinned-signer-transition-tag",
		"push-working-branch",
		"open-one-ready-pull-request",
		"run-required-public-commit-gate",
		"squash-merge-promptly-after-qa-and-security-acceptance",
		"append-public-safe-fallback-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1PRFallbackPath,
		wave1PRFallbackSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"preserve-the-rejected-signed-commit-exactly",
		"verify-every-feature-commit-with-the-pinned-signer",
		"require-exact-base-head-event-and-two-parent-PR-topology",
		"require-signed-transition-tag-tree-equal-reviewed-and-squash-main-trees",
		"require-public-gate-on-PR-and-protected-main",
		"do-not-retry-the-rejected-direct-push",
	},
	"addendum.prohibitedEffects": {
		"authorize-the-rejected-effect-retroactively",
		"retry-direct-main-push",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"move-or-delete-existing-publication-tags",
		"create-or-move-any-tag-other-than-the-transition-tag",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1MainCIFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3PRFallbackMainCIAddendum"},
	{path: "addendum.id", value: "WAVE-1-pr-fallback-main-ci-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T09:36:15Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T09:36:15Z"},
	{path: "addendum.baseCommit", value: wave1TransitionReviewedHead},
	{path: "addendum.baseTree", value: wave1TransitionReviewedTree},
	{path: "addendum.publicationBaseCommit", value: wave1PublishedMain},
	{path: "addendum.publicationBaseTree", value: wave1PublishedTree},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.workingBranch", value: wave1PRFallbackBranch},
	{path: "addendum.pullRequest", value: "5"},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.sourceDirective", value: "authorised"},
	{path: "addendum.purpose", value: "correct protected-main squash admission by binding the signed reviewed feature tree without requiring impossible commit identity"},
	{path: "addendum.priorAddendum", value: "WAVE-1-pr-fallback-addendum"},
	{path: "addendum.priorAddendumSHA256", value: "32002ec48fdc8d7c2d5c8300df9be39b5b7c944edf0b16419fdd9b2fd7561fe9"},
	{path: "addendum.priorAddendumSignatureSHA256", value: "00bad1bea742a9c63bc9ef3016e6bf887d7a1d0ab1c7f1ec2b222651f80d9d9e"},
	{path: "addendum.reviewDispositionComment", value: "01a03d6c-f79c-77e3-848c-df769ff27d48"},
	{path: "addendum.reviewedHead", value: wave1TransitionReviewedHead},
	{path: "addendum.reviewedTree", value: wave1TransitionReviewedTree},
	{path: "addendum.failureFingerprint", value: "public-pr-fallback-tag-main-squash-commit-identity-unsatisfiable"},
	{path: "addendum.priorTransitionTag", value: wave1TransitionTag},
	{path: "addendum.priorTransitionTagObject", value: wave1TransitionTagObject},
	{path: "addendum.priorTransitionTagTarget", value: wave1TransitionReviewedHead},
	{path: "addendum.priorTransitionTagTree", value: wave1TransitionReviewedTree},
	{path: "addendum.successorTransitionTag", value: wave1SuccessorTransitionTag},
	{path: "addendum.successorTransitionTagMessage", value: wave1SuccessorTagMessage},
	{path: "addendum.mergeMode", value: "squash"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.protectedMainUnchangedUntilMerge", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1MainCIFixNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-pr-fallback-main-ci-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1MainCIFixSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"create-one-pinned-signer-correction-commit",
		"preserve-the-v1-transition-tag-exactly",
		"create-and-push-one-pinned-signer-v2-transition-tag",
		"update-existing-pull-request",
		"run-required-public-commit-gate",
		"squash-merge-promptly-after-fresh-qa-and-security-acceptance",
		"append-public-safe-correction-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1MainCIFixPath,
		wave1MainCIFixSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"preserve-reviewed-head-and-v1-tag-as-immutable-history",
		"verify-every-feature-commit-with-the-pinned-signer",
		"require-v2-tag-target-equal-current-reviewed-feature-head-on-pull-request",
		"require-v2-tag-tree-equal-pull-request-and-squash-main-trees",
		"require-squash-main-commit-distinct-from-v2-tag-target",
		"require-exact-base-head-event-and-topology",
		"require-public-gate-on-updated-pull-request-and-protected-main",
	},
	"addendum.prohibitedEffects": {
		"authorize-the-failed-review-retroactively",
		"move-or-delete-the-v1-transition-tag",
		"retry-direct-main-push",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"create-or-move-any-tag-other-than-the-v2-transition-tag",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1CIFixtureFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3PRFallbackFixtureStabilizationAddendum"},
	{path: "addendum.id", value: "WAVE-1-pr-fallback-fixture-stabilization-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T09:53:55Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T09:53:55Z"},
	{path: "addendum.baseCommit", value: wave1CIFixtureReviewedHead},
	{path: "addendum.baseTree", value: wave1CIFixtureReviewedTree},
	{path: "addendum.publicationBaseCommit", value: wave1PublishedMain},
	{path: "addendum.publicationBaseTree", value: wave1PublishedTree},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.workingBranch", value: wave1PRFallbackBranch},
	{path: "addendum.pullRequest", value: "5"},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.sourceDirective", value: "authorised"},
	{path: "addendum.purpose", value: "disable Git auto-maintenance only inside disposable test repositories to eliminate post-assertion cleanup races"},
	{path: "addendum.priorAddendum", value: "WAVE-1-pr-fallback-main-ci-addendum"},
	{path: "addendum.priorAddendumSHA256", value: "8880fd546cf70fff39a02ba2851a74414f892a06741ae41f17a40532044bc130"},
	{path: "addendum.priorAddendumSignatureSHA256", value: "b757aa7fef86ec8f4f6e417f795e02d97a6ee1a0fa5f2edfd39f8d51efee4256"},
	{path: "addendum.blockedDispositionComment", value: "01a03d7d-5f5a-70e2-8ca2-93e1c76e3c02"},
	{path: "addendum.failedRun", value: "32955169341"},
	{path: "addendum.failedAttempts", value: "1,2"},
	{path: "addendum.failureFingerprint", value: "go-test-tempdir-cleanup-git-pack-race"},
	{path: "addendum.priorTransitionTag", value: wave1SuccessorTransitionTag},
	{path: "addendum.priorTransitionTagObject", value: wave1SuccessorTagObject},
	{path: "addendum.priorTransitionTagTarget", value: wave1CIFixtureReviewedHead},
	{path: "addendum.priorTransitionTagTree", value: wave1CIFixtureReviewedTree},
	{path: "addendum.successorTransitionTag", value: wave1FinalTransitionTag},
	{path: "addendum.successorTransitionTagMessage", value: wave1FinalTransitionTagMessage},
	{path: "addendum.mergeMode", value: "squash"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.protectedMainUnchangedUntilMerge", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1CIFixtureFixNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-pr-fallback-fixture-stabilization-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1CIFixtureFixSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"create-one-pinned-signer-fixture-stabilization-commit",
		"preserve-v1-and-v2-transition-tags-exactly",
		"create-and-push-one-pinned-signer-v3-transition-tag",
		"update-existing-pull-request",
		"run-one-fresh-public-commit-gate",
		"squash-merge-promptly-after-fresh-qa-and-security-acceptance",
		"append-public-safe-stabilization-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1CIFixtureFixPath,
		wave1CIFixtureFixSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"preserve-b9ca75d-and-v2-tag-as-immutable-history",
		"verify-every-feature-commit-with-the-pinned-signer",
		"disable-git-auto-maintenance-only-in-disposable-test-repositories",
		"require-v3-tag-target-equal-current-reviewed-feature-head-on-pull-request",
		"require-v3-tag-tree-equal-pull-request-and-squash-main-trees",
		"require-exact-base-head-event-and-topology",
		"require-fresh-public-gate-and-immutable-reviews",
	},
	"addendum.prohibitedEffects": {
		"rerun-the-exhausted-failed-workflow",
		"mutate-production-or-developer-git-configuration",
		"move-or-delete-v1-or-v2-transition-tags",
		"retry-direct-main-push",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"create-or-move-any-tag-other-than-the-v3-transition-tag",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1PlanningGrantSequences = map[string][]string{
	"grant.targetBeads": {
		"M3-W001",
		"M3-P001",
	},
	"grant.allowedEffects": {
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-signed-git-commits",
		"push-working-branch",
		"open-or-update-pull-request",
		"append-public-safe-contract-status-intent-and-receipt",
	},
	"grant.authorizedPaths": {
		"AGENTS.md",
		"README.md",
		".harness/docsync.yaml",
		".harness/manifest.yaml",
		".harness/grants/WAVE-1-contract-publication.yaml",
		".harness/grants/WAVE-1-contract-publication.yaml.sig",
		"docs/QUALITY_SCORE.md",
		"docs/code-documentation-map.md",
		"docs/design-docs/ADR-001-git-beads-authority.md",
		"docs/design-docs/ADR-006-local-substrate.md",
		"docs/design-docs/index.md",
		"docs/evidence/H-001-review-disposition.md",
		"docs/evidence/WAVE-1-contract-publication.md",
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/features/F-001-doctrine-foundation.md",
		"docs/features/F-002-work-authority.md",
		"docs/features/F-003-local-substrate.md",
		"docs/features/README.md",
		"docs/goals/active.md",
		"docs/product-decisions/PD-001-public-first.md",
		"docs/product-decisions/PD-002-git-beads-authority.md",
		"docs/product-decisions/PD-003-provider-neutral.md",
		"docs/product-specs/foundation.md",
		"docs/product-specs/index.md",
		"docs/product-specs/local-substrate.md",
		"docs/product-specs/work-authority.md",
		"internal/doctrine/doctrine.go",
		"internal/doctrine/doctrine_test.go",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"internal/doctrine/plan.go",
		"internal/doctrine/plan_test.go",
		"internal/doctrine/public.go",
		"internal/doctrine/public_test.go",
	},
	"grant.prohibitedEffects": {
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"trust-escalation",
		"provider-or-customer-data-access",
		"credential-persistence",
	},
	"verification.order": {
		"qa",
		"security-reviewer",
		"delivery-orchestrator",
	},
}

var wave1DispositionScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3RecoveryDispositionGrant"},
	{path: "disposition.id", value: "WAVE-1-recovery-disposition"},
	{path: "disposition.classification", value: "PUBLIC"},
	{path: "disposition.issuedAt", value: "2026-08-26T06:44:37Z"},
	{path: "disposition.expiresAt", value: "2026-08-29T06:44:37Z"},
	{path: "disposition.baseCommit", value: "fc9f6641d0f739a401a4f7be3bc0ee575df1310a"},
	{path: "disposition.workingBranch", value: wave1PlanningGrantBranch},
	{path: "disposition.signerRole", value: "human-bootstrap-authority"},
	{path: "disposition.coordinator", value: "delivery-orchestrator"},
	{path: "disposition.failureOwnership", value: "foundation"},
	{path: "disposition.purpose", value: "prospectively constrain Wave-1 recovery disposition, final description repair, and squash-safe signed-tree publication"},
	{path: "disposition.retroactiveAuthorization", value: "false"},
	{path: "disposition.priorExternalEffectsAccepted", value: "false"},
	{path: "disposition.autonomousMutation", value: "false"},
	{path: "disposition.implementationClaimed", value: "false"},
	{path: "disposition.liveLeaseAsserted", value: "false"},
	{path: "disposition.originalArtifactSHA256", value: "46a7e8459204b29e739a27859539deb44f6a54d79e4ac64ad6c3cc358d9b5d04"},
	{path: "disposition.originalDetachedSignatureSHA256", value: "4ab35549b3cb64bd58f59006aeadcbfdca0754850662481e44c0f9c8e51f5356"},
	{path: "disposition.originalArtifactGitDisposition", value: "not-committed-because-the-pinned-scanner-flags-two-public-checksum-fields"},
	{path: "disposition.stateSnapshot", value: "WAVE-1-authority-recovery-state.json"},
	{path: "disposition.stateSnapshotSHA256", value: wave1DispositionSnapshotSHA},
	{path: "disposition.stateSnapshotSchema", value: "mars3-authority-observation/v2"},
	{path: "disposition.preAuthorityRevision", value: "k46ees2p"},
	{path: "disposition.legacyStateDigestsReconstructible", value: "false"},
	{path: "disposition.p001CorrectionIdempotencyKey", value: "wave1-p001-shared-path-description-correction-v1"},
	{path: "disposition.p001PreDescriptionSHA256", value: "828e32051ee661e6d31c5017e996e3d8ec82257ca0a17e263f974248c5772314"},
	{path: "disposition.p001PostDescriptionSHA256", value: "d24074a56a4df0d150450555b981f84fa377364f40c9c9191e49f4ecae73c20c"},
	{path: "disposition.publicationMode", value: "squash-with-signed-tree-tag"},
	{path: "disposition.publicationTag", value: wave1PriorPublicationTag},
	{path: "disposition.publicationTagMessage", value: wave1PriorPublicationTagMessage},
	{path: "disposition.rulesetId", value: "21510926"},
	{path: "disposition.rulesetObservedUpdatedAt", value: "2026-08-26T03:00:36.071+01:00"},
	{path: "disposition.rulesetMutationAllowed", value: "false"},
	{path: "disposition.requiredStatusCheck", value: "Public commit gate"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.recoveryDispositionRequired", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.rawAuthorityPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1DispositionNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-recovery-disposition.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1DispositionSequences = map[string][]string{
	"disposition.targetBeads":  {"M3-W001", "M3-P001"},
	"disposition.mutableBeads": {"M3-P001"},
	"disposition.allowedEffects": {
		"verify-and-independently-disposition-observed-final-authority-state",
		"replace-m3-p001-description-with-exact-postimage",
		"append-public-safe-disposition-intent-and-receipt",
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-pinned-signer-git-commits",
		"create-and-push-one-pinned-signer-publication-tag",
		"push-working-branch",
		"open-or-update-pull-request",
	},
	"disposition.authorizedPaths": {
		".harness/docsync.yaml",
		".harness/manifest.yaml",
		wave1DispositionSnapshot,
		wave1DispositionPath,
		wave1DispositionSignature,
		"AGENTS.md",
		"README.md",
		"docs/QUALITY_SCORE.md",
		"docs/code-documentation-map.md",
		"docs/design-docs/ADR-001-git-beads-authority.md",
		"docs/design-docs/ADR-006-local-substrate.md",
		"docs/design-docs/index.md",
		"docs/evidence/H-001-review-disposition.md",
		"docs/evidence/WAVE-1-contract-publication.md",
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/features/F-001-doctrine-foundation.md",
		"docs/features/F-002-work-authority.md",
		"docs/features/F-003-local-substrate.md",
		"docs/features/README.md",
		"docs/goals/active.md",
		"docs/product-decisions/PD-001-public-first.md",
		"docs/product-decisions/PD-002-git-beads-authority.md",
		"docs/product-decisions/PD-003-provider-neutral.md",
		"docs/product-specs/foundation.md",
		"docs/product-specs/index.md",
		"docs/product-specs/local-substrate.md",
		"docs/product-specs/work-authority.md",
		"internal/doctrine/docsync.go",
		"internal/doctrine/doctrine.go",
		"internal/doctrine/doctrine_test.go",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"internal/doctrine/plan.go",
		"internal/doctrine/plan_test.go",
		"internal/doctrine/public.go",
	},
	"disposition.requiredP001Postconditions": {
		"native-status-open",
		"lifecycle-backlog",
		"claim-absent",
		"lease-absent",
		"dependencies-exactly-M3-H001-and-M3-W001",
		"exclusive-paths-unchanged",
		"all-other-Beads-unchanged",
	},
	"disposition.requiredPublicationProperties": {
		"every-feature-commit-has-the-pinned-ssh-signature",
		"signed-tag-target-descends-from-the-exact-grant-base",
		"signed-tag-target-tree-equals-reviewed-pr-head-tree",
		"squash-main-tree-equals-signed-tag-target-tree",
		"tag-keeps-reviewed-feature-history-reachable",
		"required-public-commit-gate-passes-on-pr-and-main",
	},
	"disposition.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"mutate-m3-w001",
		"mutate-p001-metadata-dependencies-lifecycle-claim-or-lease",
		"mutate-any-other-bead",
		"any-description-write-other-than-the-exact-p001-postimage",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"provider-or-customer-data-access",
		"credential-persistence",
		"add-secret-scanner-ignore-or-allowlist",
		"mutate-github-rules-or-repository-settings",
		"create-or-move-any-other-tag",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1AddendumScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3CIRecoveryAddendum"},
	{path: "addendum.id", value: "WAVE-1-ci-recovery-addendum"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T07:25:36Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T07:25:36Z"},
	{path: "addendum.baseCommit", value: wave1AddendumBase},
	{path: "addendum.workingBranch", value: wave1PlanningGrantBranch},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.purpose", value: "prospectively repair nullable GitHub pull-request merge identity admission without weakening publication controls"},
	{path: "addendum.retroactiveAuthorization", value: "false"},
	{path: "addendum.priorHead", value: wave1AddendumBase},
	{path: "addendum.priorTree", value: wave1AddendumBaseTree},
	{path: "addendum.priorTag", value: wave1PriorPublicationTag},
	{path: "addendum.priorTagObject", value: wave1PriorPublicationTagObject},
	{path: "addendum.failedRunId", value: "32941818590"},
	{path: "addendum.failedRunAttempts", value: "2"},
	{path: "addendum.finalFailedJobId", value: "98095787715"},
	{path: "addendum.failureFingerprint", value: "public.planning_grant_branch-github-pr-null-merge-commit-sha"},
	{path: "addendum.rootCause", value: "pull-request-event-merge-commit-sha-was-null"},
	{path: "addendum.correctionInvariant", value: "nullable-event-field-requires-github-sha-event-base-head-and-exact-two-parent-git-topology"},
	{path: "addendum.successorTag", value: wave1V2PublicationTag},
	{path: "addendum.successorTagMessage", value: wave1V2PublicationTagMessage},
	{path: "addendum.v1TagImmutable", value: "true"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.implementationClaimed", value: "false"},
	{path: "addendum.liveLeaseAsserted", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.failedAttemptsPreserved", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.rawRunnerPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1AddendumNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-ci-recovery-addendum.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1AddendumSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-pinned-signer-git-commits",
		"create-and-push-one-pinned-signer-retry-tag",
		"push-working-branch",
		"update-existing-pull-request",
		"append-public-safe-recovery-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1AddendumPath,
		wave1AddendumSignature,
		"docs/evidence/WAVE-1-contract-publication.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredCorrectionProperties": {
		"accept-null-event-merge-sha-only-for-verified-pull-request-merge",
		"reject-nonempty-mismatched-event-merge-sha",
		"require-github-sha-equals-checked-out-merge",
		"require-event-base-and-head-equal-git-merge-parents",
		"retain-hosted-runner-workflow-job-repository-and-workspace-facts",
		"retain-pinned-workflow-digest-and-signed-feature-commit-validation",
		"enforce-addendum-only-paths-after-the-v1-target",
		"preserve-v1-tag-and-record-both-failed-attempts",
		"require-v2-tag-tree-equal-updated-review-and-squash-main-trees",
	},
	"addendum.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"move-or-delete-v1-tag",
		"create-or-move-any-tag-other-than-v2",
		"accept-nonempty-mismatched-event-merge-sha",
		"use-nullable-merge-sha-rule-outside-pull-request-ci",
		"mutate-workflow-ruleset-or-repository-settings",
		"add-secret-scanner-ignore-or-allowlist",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var wave1V3AddendumScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3CIRecoveryAddendumV3"},
	{path: "addendum.id", value: "WAVE-1-ci-recovery-addendum-v3"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T07:56:09Z"},
	{path: "addendum.expiresAt", value: "2026-08-29T07:56:09Z"},
	{path: "addendum.baseCommit", value: wave1V3AddendumBase},
	{path: "addendum.baseTree", value: wave1V3AddendumBaseTree},
	{path: "addendum.workingBranch", value: wave1PlanningGrantBranch},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.purpose", value: "prospectively admit advisory stale GitHub pull-request merge metadata without weakening canonical checkout and topology controls"},
	{path: "addendum.retroactiveAuthorization", value: "false"},
	{path: "addendum.priorHead", value: wave1V3AddendumBase},
	{path: "addendum.priorTree", value: wave1V3AddendumBaseTree},
	{path: "addendum.v1Tag", value: wave1PriorPublicationTag},
	{path: "addendum.v1TagObject", value: wave1PriorPublicationTagObject},
	{path: "addendum.v1TagTarget", value: wave1AddendumBase},
	{path: "addendum.v2Tag", value: wave1V2PublicationTag},
	{path: "addendum.v2TagObject", value: wave1V2PublicationTagObject},
	{path: "addendum.v2TagTarget", value: wave1V3AddendumBase},
	{path: "addendum.failedRunId", value: "32943782330"},
	{path: "addendum.failedRunAttempts", value: "2"},
	{path: "addendum.finalFailedJobId", value: "98101129557"},
	{path: "addendum.observedCheckout", value: "3ffd69c107f1492883b65a67678f7239602299a4"},
	{path: "addendum.observedBase", value: wave1PlanningGrantBase},
	{path: "addendum.observedHead", value: wave1V3AddendumBase},
	{path: "addendum.observedStalePayloadMerge", value: wave1V3ObservedStaleMerge},
	{path: "addendum.failureFingerprint", value: "public.planning_grant_branch-github-pr-stale-nonempty-merge-commit-sha"},
	{path: "addendum.rootCause", value: "pull-request-event-payload-retained-the-prior-test-merge-identity-after-the-head-advanced"},
	{path: "addendum.correctionInvariant", value: "payload-merge-identity-is-advisory-while-github-sha-event-base-head-signed-tree-and-exact-two-parent-topology-remain-authoritative"},
	{path: "addendum.successorTag", value: wave1PublicationTag},
	{path: "addendum.successorTagMessage", value: wave1PublicationTagMessage},
	{path: "addendum.v1TagImmutable", value: "true"},
	{path: "addendum.v2TagImmutable", value: "true"},
	{path: "addendum.autonomousMutation", value: "false"},
	{path: "addendum.implementationClaimed", value: "false"},
	{path: "addendum.liveLeaseAsserted", value: "false"},
	{path: "addendum.canonicalWorkMutationAllowed", value: "false"},
	{path: "addendum.githubPolicyMutationAllowed", value: "false"},
	{path: "addendum.secretScannerExceptionAllowed", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.failedAttemptsPreserved", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.rawRunnerPayloadIncluded", value: "false"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: wave1V3AddendumNamespace},
	{path: "integrity.detachedSignature", value: "WAVE-1-ci-recovery-addendum-v3.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var wave1V3AddendumSequences = map[string][]string{
	"addendum.allowedEffects": {
		"edit-listed-public-paths",
		"run-public-commit-gate",
		"create-pinned-signer-git-commits",
		"create-and-push-one-pinned-signer-v3-tag",
		"push-working-branch",
		"update-existing-pull-request",
		"append-public-safe-recovery-intent-and-receipt",
	},
	"addendum.authorizedPaths": {
		wave1V3AddendumPath,
		wave1V3AddendumSignature,
		"docs/evidence/WAVE-1-contract-publication.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredCorrectionProperties": {
		"treat-optional-payload-merge-sha-as-advisory-only-in-pull-request-ci",
		"accept-absent-null-current-or-stale-lowercase-forty-hex-payload-merge-sha",
		"reject-malformed-nonempty-payload-merge-identity",
		"require-github-sha-equals-checked-out-merge",
		"require-event-base-and-head-equal-git-merge-parents-in-order",
		"retain-hosted-runner-workflow-job-repository-workspace-and-workflow-digest-facts",
		"retain-signed-feature-history-phase-specific-path-and-publication-tree-validation",
		"enforce-v3-addendum-only-paths-after-the-v2-target",
		"preserve-v1-and-v2-tags-and-record-all-failed-attempts",
		"require-v3-tag-tree-equal-updated-review-and-squash-main-trees",
	},
	"addendum.prohibitedEffects": {
		"authorize-prior-effects-retroactively",
		"move-or-delete-v1-or-v2-tag",
		"create-or-move-any-tag-other-than-v3",
		"use-advisory-merge-identity-rule-outside-pull-request-ci",
		"accept-malformed-payload-merge-identity",
		"weaken-github-sha-event-base-head-or-exact-two-parent-topology-validation",
		"mutate-workflow-ruleset-or-repository-settings",
		"add-secret-scanner-ignore-or-allowlist",
		"mutate-any-bead-work-field-or-dependency",
		"claim-or-transition-bead",
		"issue-or-assert-live-lease",
		"runtime-or-platform-implementation",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001BootstrapGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3ImplementationBootstrapGrant"},
	{path: "grant.id", value: "W-001-bootstrap"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T10:30:00Z"},
	{path: "grant.expiresAt", value: "2026-08-29T10:30:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001BootstrapBase},
	{path: "grant.baseTree", value: w001BootstrapBaseTree},
	{path: "grant.workingBranch", value: w001BootstrapBranch},
	{path: "grant.reviewTag", value: w001BootstrapReviewTag},
	{path: "grant.reviewTagMessage", value: w001BootstrapReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.blockerComment", value: "01a03d98-a2e2-77db-bbd6-76575366454f"},
	{path: "grant.purpose", value: "publish and execute one reviewed atomic W-001 bootstrap claim without asserting a live lease"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.profile", value: "work-authority-engineer"},
	{path: "grant.attemptId", value: "w001-bootstrap-3135f1d1-b0d4-4956-9fc9-1852310bfd77"},
	{path: "grant.replayRef", value: "w001-claim-97a7e81e-e749-4cb0-8a0a-8e269e13e38f"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.productionEffects", value: "false"},
	{path: "expected.beadsProject", value: "e9669a62-5be6-4b94-85f8-c502c29d394a"},
	{path: "expected.nativeStatus", value: "open"},
	{path: "expected.lifecycleState", value: "backlog"},
	{path: "expected.claimState", value: "unclaimed"},
	{path: "expected.assignee", value: "work-authority-engineer"},
	{path: "expected.createdAt", value: "2026-08-26T05:09:05Z"},
	{path: "expected.updatedAt", value: "2026-08-26T06:22:03Z"},
	{path: "expected.metadataSHA256", value: "10c61003cb39518f57905620fcc0c47d29950fe82ae8d98a3111a057fa554dba"},
	{path: "expected.labelsSHA256", value: "be506df06d8c206a3919a71a57e8aaacd2b5e1e233e25bafc2f5f87f306b188c"},
	{path: "expected.dependency", value: "M3-H001"},
	{path: "expected.dependencyType", value: "blocks"},
	{path: "expected.dependencyNativeStatus", value: "closed"},
	{path: "expected.dependencyLifecycleState", value: "done"},
	{path: "expected.dependencyDigest", value: "3ad0bca78b14e4e1fd5544477f131c0a86dd8a4d4e9563d43fa4ae1c202f4100"},
	{path: "expected.lineageDigest", value: "9f3e91b4b642dc740898347c35e8f38abc35cc3ac1be83c81fe122cc308eaced"},
	{path: "postimage.nativeStatus", value: "in_progress"},
	{path: "postimage.lifecycleState", value: "in-progress"},
	{path: "postimage.claimState", value: "claimed"},
	{path: "postimage.removeLabel", value: "backlog"},
	{path: "postimage.addLabel", value: "in-progress"},
	{path: "postimage.generationRef", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "postimage.issueIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "postimage.issueMutationSequence", value: "1"},
	{path: "postimage.dependencyGraphRevision", value: "1"},
	{path: "postimage.metadataSHA256", value: "3a0bbed5ca93acf77d04eedfad6bcaa16a9701f3e6da3f8669eb9928f6d09139"},
	{path: "postimage.labelsSHA256", value: "3e4e77e20ee7a46dd77c4a9884dee51aa9f0fa9f2445099a0cb457d72cb83bbb"},
	{path: "postimage.metadataBase64", value: "eyJib290c3RyYXBDbGFpbSI6eyJhdHRlbXB0SWQiOiJ3MDAxLWJvb3RzdHJhcC0zMTM1ZjFkMS1iMGQ0LTQ5NTYtOWZjOS0xODUyMzEwYmZkNzciLCJiYXNlQ29tbWl0IjoiMzdiNTViOTEyYjIwNzE1MzQ5YmM1MGUwNTI0Yzg1ZDRiMjJmMTc3MiIsImdyYW50SWQiOiJXLTAwMS1ib290c3RyYXAiLCJpZGVtcG90ZW5jeUtleSI6IncwMDEtY2xhaW0tOTdhN2U4MWUtZTc0OS00Y2IwLThhMGEtOGUyNjllMTNlMzhmIn0sImNvbnRyYWN0UHVibGljYXRpb25SZXF1aXJlZCI6dHJ1ZSwiY29vcmRpbmF0b3IiOiJkZWxpdmVyeS1vcmNoZXN0cmF0b3IiLCJkaXNwbGF5SWQiOiJXLTAwMSIsImV4Y2x1c2l2ZVBhdGhzIjpbImludGVybmFsL2F1dGhvcml0eS8qKiIsImNtZC9tYXJzMy1hdXRob3JpdHkvKioiLCJhcGkvYXV0aG9yaXR5LyoqIiwiZGF0YWJhc2UvYXV0aG9yaXR5LyoqIiwiZGVwbG95L2F1dGhvcml0eS8qKiIsImRvY3MvZXZpZGVuY2UvVy0wMDEtdmFsaWRhdGlvbi5tZCIsImdvLm1vZCIsImdvLnN1bSIsIk1ha2VmaWxlIiwiTk9USUNFIiwiVEhJUkRfUEFSVFlfTk9USUNFUyJdLCJmYWlsdXJlT3duZXJzaGlwIjoiZm91bmRhdGlvbiIsImZlYXR1cmVJZCI6IkYtMDAyIiwiZ29hbElkcyI6WyJHLTAwMSJdLCJsaWZlY3ljbGVTdGF0ZSI6ImluLXByb2dyZXNzIiwicHJvZHVjdERlY2lzaW9uSWRzIjpbIlBELTAwMiJdLCJwdWJsaWNEaXNjbG9zdXJlIjp0cnVlLCJyaXNrIjoiY3JpdGljYWwiLCJzY2VuYXJpb0lkcyI6WyJGLTAwMi1TMSIsIkYtMDAyLVMyIiwiRi0wMDItUzMiLCJGLTAwMi1TNCIsIkYtMDAyLVM1IiwiRi0wMDItUzYiXSwic2NoZW1hVmVyc2lvbiI6MSwidmVyaWZpY2F0aW9uT3JkZXIiOlsicWEiLCJzZWN1cml0eS1yZXZpZXdlciIsImRlbGl2ZXJ5LW9yY2hlc3RyYXRvciJdLCJ3b3JrVHlwZSI6ImVuYWJsZXIiLCJ3b3JrVmVyc2lvbiI6eyJhdXRob3JpdHlHZW5lcmF0aW9uIjoiNmU3OWZmODEtYTAwNy00MmE1LWExNzgtN2NlNThkYmI3MThiIiwiZGVwZW5kZW5jeUdyYXBoUmV2aXNpb24iOjEsImlzc3VlSW5jYXJuYXRpb24iOiJlMWU4ZDJkM2Y4MDg3MTA5NmE1NjhmYjQ4OWY0OTU3NWE0MmFiZDM3YjI2OWRmOWZhZjc3N2EwOWNkNjg5YjQxIiwiaXNzdWVNdXRhdGlvblNlcXVlbmNlIjoxfX0"},
	{path: "toolchain.beadsVersion", value: "1.2.2"},
	{path: "toolchain.beadsSourceCommit", value: beadsCommit},
	{path: "toolchain.beadsBinarySHA256", value: "6cc5cf1d3fea5774606af82410ac05e35b78ad5f404f1da5be33c93ff087cffb"},
	{path: "toolchain.doltSourceCommit", value: doltCommit},
	{path: "toolchain.doltModule", value: "github.com/dolthub/dolt/go v0.40.5-0.20260605230755-1bf533220ab0"},
	{path: "toolchain.doltModuleSHA256", value: "oPg5f5bYFy5x7Ws2qtVG7wiva96cIh9SFg7nrC4n7QA="},
	{path: "toolchain.goVersion", value: "go1.26.2"},
	{path: "toolchain.goOS", value: "darwin"},
	{path: "toolchain.goArch", value: "arm64"},
	{path: "toolchain.goBinarySHA256", value: "005640c7ff93028cb704283b0f737f2db3faf8b51b2561170c769b83905da646"},
	{path: "toolchain.cgoEnabled", value: "true"},
	{path: "toolchain.icuFormula", value: "icu4c@78 78.2"},
	{path: "toolchain.doltTestImage", value: "dolthub/dolt-sql-server@sha256:6b651663c5024d98a98a4db7226a5e85f90a9344c78fee85617c0fb4a30c6e64"},
	{path: "toolchain.ryukDisabled", value: "true"},
	{path: "toolchain.patchPath", value: "internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"},
	{path: "toolchain.patchSHA256", value: "50128252828352366ced6560371468a5746c2603ef89ea746a33be8994ffceb6"},
	{path: "toolchain.patchedBinarySHA256", value: "949e1d535e19ecb39e974b90b7321ef1f7f7d6b77c3958d72edb07e78d9def5a"},
	{path: "toolchain.helperCommandPath", value: "cmd/mars3-authority/main.go"},
	{path: "toolchain.helperCommandSHA256", value: "d8ae9fcf5b04902fa3f2ece3369688ca7abf1e55f0cd4f57a611006a861979ea"},
	{path: "toolchain.helperLibraryPath", value: "internal/authority/bootstrap/bootstrap.go"},
	{path: "toolchain.helperLibrarySHA256", value: "d039c787f73e98f059937242e068d76c12753cc9accedc025bf619e1fa63c0fd"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequiredBeforeClaim", value: "true"},
	{path: "verification.externalStateReadbackRequired", value: "true"},
	{path: "verification.disposableAtomicityConformanceRequired", value: "true"},
	{path: "verification.postReviewExecutionAuthorizationRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001BootstrapGrantNamespace},
	{path: "integrity.detachedSignature", value: "W-001-bootstrap.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001BootstrapGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"edit-exact-preclaim-paths",
		"build-patched-Beads-from-the-exact-signed-source-and-patch",
		"run-disposable-atomic-claim-conformance",
		"create-pinned-signer-feature-commits-and-one-signed-review-tag",
		"push-one-feature-branch-and-review-tag-and-open-one-ready-PR",
		"run-public-commit-gate-and-obtain-independent-QA-and-Security-review",
		"squash-merge-promptly-with-reviewed-tree-equality",
		"append-public-safe-authority-intents-receipts-and-dispositions",
		"execute-one-expected-preimage-CAS-claim-on-accepted-main",
		"append-one-public-safe-canonical-claim-receipt",
	},
	"grant.preclaimPaths": {
		w001BootstrapGrantPath,
		w001BootstrapGrantSignature,
		"cmd/mars3-authority/main.go",
		"internal/authority/bootstrap/bootstrap.go",
		"internal/authority/bootstrap/bootstrap_test.go",
		"internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"docs/design-docs/ADR-001-git-beads-authority.md",
		"docs/product-specs/work-authority.md",
		"docs/features/F-002-work-authority.md",
		"docs/evidence/W-001-bootstrap-transition.md",
		"NOTICE",
		"THIRD_PARTY_NOTICES",
	},
	"grant.implementationPaths": {
		"internal/authority/**", "cmd/mars3-authority/**", "api/authority/**",
		"database/authority/**", "deploy/authority/**", "docs/evidence/W-001-validation.md",
		"go.mod", "go.sum", "Makefile", "NOTICE", "THIRD_PARTY_NOTICES",
	},
	"grant.prohibitedEffects": {
		"execute-claim-before-helper-merge-protected-main-success-QA-and-Security-acceptance",
		"execute-claim-without-a-separate-signed-short-lived-post-review-authorization",
		"use-raw-SQL-hidden-code-or-a-two-transaction-claim",
		"mutate-another-Bead-dependency-or-authority-project",
		"issue-or-assert-a-live-lease-before-verified-gateway-issuance",
		"edit-outside-the-current-signed-phase-paths",
		"move-delete-or-reuse-the-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"reconcile-plan-or-manifest-without-a-separate-signed-postclaim-grant",
		"begin-gateway-implementation-under-this-bootstrap-claim-grant",
		"autonomous-mutation", "trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimReconciliationGrant"},
	{path: "grant.id", value: "W-001-postclaim-reconciliation"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T20:33:58Z"},
	{path: "grant.expiresAt", value: "2026-08-28T20:33:58Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimBase},
	{path: "grant.baseTree", value: w001PostclaimBaseTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimReviewTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "reconcile the verified canonical W-001 claim into Git without a lease or implementation authority"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorGrant", value: "W-001-bootstrap"},
	{path: "grant.priorGrantSHA256", value: "37b4b0df773fa6bace2a06af34ff546e12c9e2f42eb8bb5865f1b9405727e34d"},
	{path: "grant.priorGrantSignatureSHA256", value: "eeba6098de123f4d9d47c33d4349015775591511784d6c29b709ed1d45927e15"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "claim.attemptId", value: "w001-bootstrap-3135f1d1-b0d4-4956-9fc9-1852310bfd77"},
	{path: "claim.replayReferenceSHA256", value: "321df42aa6cd67c3ae42b687b16927d81fc19b42e124819bd266b84b22c2d1a0"},
	{path: "claim.executionAuthorizationPayloadSHA256", value: "2a2428537a5e42ed6f69afaa42771b9b404705e5593b554129c6ce45f5aec297"},
	{path: "claim.executionAuthorizationSignatureSHA256", value: "d7bc5a1c824bde6ee72d755660afccaad569840b2db29971bf3b8b087540ce5c"},
	{path: "claim.receiptSHA256", value: "04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126"},
	{path: "claim.previousDoltCommit", value: "mm6m0b4655k5gpt5eren5cfkgvjtabsm"},
	{path: "claim.doltCommit", value: "67hmen0cmq0he08n7ujlqpcsmmi94fhb"},
	{path: "claim.nativeStatus", value: "in_progress"},
	{path: "claim.lifecycleState", value: "in-progress"},
	{path: "claim.claimed", value: "true"},
	{path: "claim.updatedAt", value: "2026-08-26T20:23:37Z"},
	{path: "claim.metadataSHA256", value: "3a0bbed5ca93acf77d04eedfad6bcaa16a9701f3e6da3f8669eb9928f6d09139"},
	{path: "claim.labelsSHA256", value: "3e4e77e20ee7a46dd77c4a9884dee51aa9f0fa9f2445099a0cb457d72cb83bbb"},
	{path: "claim.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "claim.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "claim.issueMutationSequence", value: "1"},
	{path: "claim.dependencyGraphRevision", value: "1"},
	{path: "claim.liveLeaseAsserted", value: "false"},
	{path: "publication.helperFeatureCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "publication.helperFeatureTree", value: w001PostclaimBaseTree},
	{path: "publication.helperReviewTag", value: w001BootstrapReviewTag},
	{path: "publication.helperReviewTagObject", value: "6409b5daecb472b415dec60b96b12bfffa3a4cd0"},
	{path: "publication.helperPullRequest", value: "6"},
	{path: "publication.helperPullRequestRun", value: "33005091777"},
	{path: "publication.mergedCommit", value: w001PostclaimBase},
	{path: "publication.mergedTree", value: w001PostclaimBaseTree},
	{path: "publication.protectedMainRun", value: "33006386000"},
	{path: "publication.qaReviewedCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "publication.qaDisposition", value: "accepted"},
	{path: "publication.securityReviewedCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "publication.securityDisposition", value: "accepted"},
	{path: "postimage.manifestSHA256", value: "ee37f6262e714aa1f853fe7b81ce9b50a1d299482dde905262e80cbb133d13e6"},
	{path: "postimage.activePlanSHA256", value: "d5ac48639cd8bd98ce57e1f0f91ddd4c358498eddb45c8255c0ca67ab56abbbc"},
	{path: "postimage.evidenceSHA256", value: "93dd425ef42e6c929a97d20e08b9d75c339b45303975162cc863dad3764dc1a0"},
	{path: "postimage.planPhase", value: "delivery"},
	{path: "postimage.currentBeadState", value: "in-progress"},
	{path: "postimage.currentBeadClaimed", value: "true"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimGrantNamespace},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-reconciliation.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-signed-postclaim-grant",
		"edit-the-exact-authorized-Git-paths",
		"bind-the-exact-canonical-claim-receipt-digest-and-bounded-public-summary",
		"reconcile-the-active-plan-to-delivery-and-W-001-in-progress",
		"reconcile-the-manifest-to-claimed-in-progress-delivery",
		"create-pinned-signer-commits-and-one-signed-review-tag",
		"push-one-review-branch-and-tag-and-open-one-ready-PR",
		"run-public-gates-and-obtain-independent-QA-and-Security-review",
		"squash-merge-promptly-with-reviewed-tree-equality",
	},
	"grant.authorizedPaths": {
		w001PostclaimGrantPath,
		w001PostclaimGrantSignature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-bootstrap-transition.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"canonical-Beads-state-remains-the-work-authority",
		"claim-receipt-binds-the-signed-execution-authorization-and-accepted-helper",
		"plan-and-manifest-postimages-exactly-match-the-signed-digests",
		"every-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-feature-commit-and-review-tag-use-the-pinned-signer",
		"reviewed-feature-tree-equals-the-protected-main-squash-tree",
		"no-live-lease-or-implementation-authority-is-created",
	},
	"grant.prohibitedEffects": {
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"edit-gateway-runtime-platform-or-product-implementation",
		"change-feature-contract-product-decision-goal-or-scenario-schedule",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimCIFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimCIStabilizationAddendum"},
	{path: "addendum.id", value: "W-001-postclaim-ci-stabilization-v2"},
	{path: "addendum.classification", value: "PUBLIC"},
	{path: "addendum.issuedAt", value: "2026-08-26T20:53:07Z"},
	{path: "addendum.expiresAt", value: "2026-08-28T20:53:07Z"},
	{path: "addendum.repository", value: planningGrantRepository},
	{path: "addendum.baseCommit", value: w001PostclaimCIFixBase},
	{path: "addendum.baseTree", value: w001PostclaimCIFixBaseTree},
	{path: "addendum.workingBranch", value: w001PostclaimBranch},
	{path: "addendum.reviewTag", value: w001PostclaimCIFixReviewTag},
	{path: "addendum.reviewTagMessage", value: w001PostclaimCIFixTagMessage},
	{path: "addendum.signerRole", value: "human-bootstrap-authority"},
	{path: "addendum.coordinator", value: "delivery-orchestrator"},
	{path: "addendum.failureOwnership", value: "foundation"},
	{path: "addendum.purpose", value: "isolate repository-local Git fixtures from inherited GitHub runner identity without weakening real checkout admission"},
	{path: "addendum.priorGrant", value: "W-001-postclaim-reconciliation"},
	{path: "addendum.priorGrantSHA256", value: "7fb4b9e6aa65661bef80887b95225973e556fe8dbc4cf77fb66aaa0f10da5dfe"},
	{path: "addendum.priorGrantSignatureSHA256", value: "3afc9d46c8c9ea436246d3bd33c2228e0afc1c1c27ceb6ff44114c761e029edd"},
	{path: "addendum.priorReviewTag", value: w001PostclaimReviewTag},
	{path: "addendum.priorReviewTagObject", value: w001PostclaimV1TagObject},
	{path: "addendum.priorReviewTagTarget", value: w001PostclaimCIFixBase},
	{path: "addendum.failedRun", value: "33012491197"},
	{path: "addendum.failedJob", value: "98322099024"},
	{path: "addendum.failureFingerprint", value: "go-test-fixtures-inherited-github-actions-environment"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimCIFixNamespace},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-ci-stabilization-v2.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimCIFixSequences = map[string][]string{
	"addendum.allowedEffects": {
		"create-and-verify-this-signed-CI-stabilization-addendum",
		"clear-GitHub-runner-environment-only-inside-repository-local-Git-fixtures",
		"add-the-exact-regression-for-the-preserved-CI-failure",
		"preserve-production-GitHub-checkout-identity-enforcement",
		"create-pinned-signer-correction-commits-and-one-signed-v2-review-tag",
		"push-the-existing-review-branch-and-v2-tag-and-rerun-the-ready-PR",
		"obtain-independent-QA-and-Security-review-before-squash-merge",
	},
	"addendum.authorizedPaths": {
		w001PostclaimCIFixPath,
		w001PostclaimCIFixSignature,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"addendum.requiredProperties": {
		"v1-review-tag-object-and-target-remain-immutable",
		"v1-failed-run-remains-public-and-unchanged",
		"fixture-environment-isolation-is-test-only",
		"real-GitHub-checkout-admission-still-requires-canonical-runner-identity",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v2-review-tag-use-the-pinned-signer",
		"reviewed-v2-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"addendum.prohibitedEffects": {
		"move-delete-or-reuse-the-v1-review-tag",
		"edit-plan-manifest-evidence-feature-or-product-contracts",
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"edit-gateway-runtime-platform-or-product-implementation",
		"weaken-production-GitHub-runner-event-topology-workflow-or-tag-validation",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimSecurityFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimSecurityCorrectionGrant"},
	{path: "grant.id", value: "W-001-postclaim-security-correction-v3"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T21:21:01Z"},
	{path: "grant.expiresAt", value: "2026-08-28T21:21:01Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimSecurityFixBase},
	{path: "grant.baseTree", value: w001PostclaimSecurityFixTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimSecurityFixTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimSecurityFixTagMsg},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "correct the superseded W-001 helper Security disposition and bind effective direct embedded M3 authority without mutating canonical work"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorAddendum", value: "W-001-postclaim-ci-stabilization-v2"},
	{path: "grant.priorAddendumSHA256", value: "526f73a4200fc294bb8cf5ad82bb4e12f6259d1490a8c98316da1ddb44edabef"},
	{path: "grant.priorAddendumSignatureSHA256", value: "4ad7b094a2328eedd0f24ae671e28a5797026b7bb8f14d89cab4829cc70d244d"},
	{path: "grant.priorReviewTag", value: w001PostclaimCIFixReviewTag},
	{path: "grant.priorReviewTagObject", value: w001PostclaimV2TagObject},
	{path: "grant.priorReviewTagTarget", value: w001PostclaimSecurityFixBase},
	{path: "grant.successfulRun", value: "33013185662"},
	{path: "grant.successfulJob", value: "98324496309"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "finding.helperFeatureCommit", value: "663d19bf190f9e3bd27edc96ee08acaa6778c853"},
	{path: "finding.helperFeatureTree", value: w001PostclaimBaseTree},
	{path: "finding.affectedPostclaimHead", value: w001PostclaimSecurityFixBase},
	{path: "finding.affectedPostclaimTree", value: w001PostclaimSecurityFixTree},
	{path: "finding.priorSecurityDisposition", value: "accepted"},
	{path: "finding.currentSecurityDisposition", value: "changes-requested"},
	{path: "finding.priorSecurityAcceptanceStatus", value: "superseded"},
	{path: "finding.failureFingerprint", value: "bootstrap-effective-database-selector-splice"},
	{path: "finding.failureClass", value: "foundation-owned"},
	{path: "finding.findingScope", value: "helper-admission-and-transaction-authority"},
	{path: "canonicalEffect.bead", value: "M3-W001"},
	{path: "canonicalEffect.nativeStatus", value: "in_progress"},
	{path: "canonicalEffect.lifecycleState", value: "in-progress"},
	{path: "canonicalEffect.assignee", value: "work-authority-engineer"},
	{path: "canonicalEffect.updatedAt", value: "2026-08-26T20:23:37Z"},
	{path: "canonicalEffect.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalEffect.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalEffect.issueMutationSequence", value: "1"},
	{path: "canonicalEffect.dependencyGraphRevision", value: "1"},
	{path: "canonicalEffect.claimReceiptSHA256", value: "04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126"},
	{path: "canonicalEffect.doltCommit", value: "67hmen0cmq0he08n7ujlqpcsmmi94fhb"},
	{path: "canonicalEffect.independentlyReadBackFromM3", value: "true"},
	{path: "canonicalEffect.alternateDatabaseUseObservedInCanonicalEffect", value: "false"},
	{path: "canonicalEffect.liveLeaseAsserted", value: "false"},
	{path: "materials.helperLibraryPath", value: "internal/authority/bootstrap/bootstrap.go"},
	{path: "materials.helperLibrarySHA256", value: w001PostclaimSecurityHelperSHA},
	{path: "materials.helperTestPath", value: "internal/authority/bootstrap/bootstrap_test.go"},
	{path: "materials.helperTestSHA256", value: w001PostclaimSecurityTestSHA},
	{path: "materials.basePatchPath", value: "internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"},
	{path: "materials.basePatchSHA256", value: w001PostclaimSecurityBasePatchSHA},
	{path: "materials.securityPatchPath", value: w001PostclaimSecurityPatchPath},
	{path: "materials.securityPatchSHA256", value: w001PostclaimSecurityPatchSHA},
	{path: "materials.patchedBinarySHA256", value: w001PostclaimSecurityBinarySHA},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimSecurityFixNS},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-security-correction-v3.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimSecurityFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-signed-Security-correction-grant",
		"record-the-additive-Security-changes-requested-and-supersession-evidence",
		"reject-or-bind-Beads-selector-files-and-force-effective-direct-embedded-M3",
		"require-same-transaction-embedded-M3-authority-before-any-bootstrap-read-or-write",
		"remove-server-transaction-bootstrap-claim-capability",
		"add-alternate-database-server-backend-and-configuration-race-regressions",
		"edit-only-the-exact-authorized-Git-paths",
		"preserve-the-observed-canonical-M3-postimage-as-effect-evidence-not-helper-acceptance",
		"create-pinned-signer-correction-commits-and-one-signed-v3-review-tag",
		"push-the-existing-review-branch-and-v3-tag-and-rerun-the-ready-PR",
		"obtain-fresh-independent-QA-and-Security-review-before-squash-merge",
	},
	"grant.authorizedPaths": {
		w001PostclaimSecurityFixPath,
		w001PostclaimSecurityFixSig,
		"docs/evidence/W-001-validation.md",
		"internal/authority/bootstrap/bootstrap.go",
		"internal/authority/bootstrap/bootstrap_test.go",
		"internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch",
		w001PostclaimSecurityPatchPath,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"prior-v1-and-v2-grants-signatures-tags-runs-and-history-remain-immutable",
		"prior-helper-Security-acceptance-is-explicitly-superseded",
		"actual-canonical-M3-postimage-remains-valid-effect-evidence-only",
		"helper-selection-and-transaction-authority-both-require-direct-embedded-M3",
		"selector-file-or-effective-database-change-fails-before-authority-state-mutation",
		"server-backed-transactions-cannot-implement-bootstrap-claim-authority",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v3-review-tag-use-the-pinned-signer",
		"reviewed-v3-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"grant.prohibitedEffects": {
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"execute-or-replay-the-canonical-W-001-bootstrap-claim",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"begin-gateway-runtime-platform-or-product-implementation",
		"edit-plan-manifest-feature-product-goal-or-scenario-contracts",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimHookFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimHookIsolationGrant"},
	{path: "grant.id", value: "W-001-postclaim-hook-isolation-v4"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T23:45:00Z"},
	{path: "grant.expiresAt", value: "2026-08-28T23:45:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimHookFixBase},
	{path: "grant.baseTree", value: w001PostclaimHookFixTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimHookFixTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimHookFixTagMsg},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "isolate the one-shot W-001 bootstrap helper from workspace hooks and last-merged local configuration without mutating canonical work"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorGrant", value: "W-001-postclaim-security-correction-v3"},
	{path: "grant.priorGrantSHA256", value: "6ce07a2b1b42b5d3fb3a31b1ec6c71cb80851bd732e52fade781d7c88b0e6caa"},
	{path: "grant.priorGrantSignatureSHA256", value: "df9dc6d62b70b40e5b0000aa10abf48afcf53fcfed581311f2bf0c61d532dfcf"},
	{path: "grant.priorReviewTag", value: w001PostclaimSecurityFixTag},
	{path: "grant.priorReviewTagObject", value: w001PostclaimV3TagObject},
	{path: "grant.priorReviewTagTarget", value: w001PostclaimHookFixBase},
	{path: "grant.successfulRun", value: "33018302554"},
	{path: "grant.successfulJob", value: "98342126652"},
	{path: "grant.qaDisposition", value: "accepted"},
	{path: "grant.securityDisposition", value: "changes-requested"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "finding.affectedPostclaimHead", value: w001PostclaimHookFixBase},
	{path: "finding.affectedPostclaimTree", value: w001PostclaimHookFixTree},
	{path: "finding.qaDisposition", value: "accepted"},
	{path: "finding.securityDisposition", value: "changes-requested"},
	{path: "finding.failureFingerprint", value: "bootstrap-workspace-hook-postcommit-effect"},
	{path: "finding.failureClass", value: "foundation-owned"},
	{path: "finding.findingScope", value: "helper-hook-and-local-configuration-isolation"},
	{path: "finding.canonicalWorkspaceAffected", value: "false"},
	{path: "canonicalEffect.bead", value: "M3-W001"},
	{path: "canonicalEffect.nativeStatus", value: "in_progress"},
	{path: "canonicalEffect.lifecycleState", value: "in-progress"},
	{path: "canonicalEffect.assignee", value: "work-authority-engineer"},
	{path: "canonicalEffect.updatedAt", value: "2026-08-26T20:23:37Z"},
	{path: "canonicalEffect.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalEffect.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalEffect.issueMutationSequence", value: "1"},
	{path: "canonicalEffect.dependencyGraphRevision", value: "1"},
	{path: "canonicalEffect.claimReceiptSHA256", value: "04cef4e421a34e0908d392fc794181db3ddb754a134e34599fa41a520c78d126"},
	{path: "canonicalEffect.doltCommit", value: "67hmen0cmq0he08n7ujlqpcsmmi94fhb"},
	{path: "canonicalEffect.independentlyReadBackFromM3", value: "true"},
	{path: "canonicalEffect.workspaceHookUseObservedInCanonicalEffect", value: "false"},
	{path: "canonicalEffect.liveLeaseAsserted", value: "false"},
	{path: "materials.validationEvidencePath", value: "docs/evidence/W-001-validation.md"},
	{path: "materials.validationEvidenceSHA256", value: "da9533a215fa2f255c242f369cbc0e9818f0c89c7f18281328633d68403aea0f"},
	{path: "materials.transitionEvidencePath", value: "docs/evidence/W-001-bootstrap-transition.md"},
	{path: "materials.transitionEvidenceSHA256", value: "ae4fdd2e0c0d03aee305be8cd6b0884cd6b98b597fd865f90c644e47d0130edc"},
	{path: "materials.helperLibraryPath", value: "internal/authority/bootstrap/bootstrap.go"},
	{path: "materials.helperLibrarySHA256", value: w001PostclaimHookHelperSHA},
	{path: "materials.helperTestPath", value: "internal/authority/bootstrap/bootstrap_test.go"},
	{path: "materials.helperTestSHA256", value: w001PostclaimHookTestSHA},
	{path: "materials.basePatchPath", value: "internal/authority/bootstrap/beads-v1.2.2-atomic-claim.patch"},
	{path: "materials.basePatchSHA256", value: w001PostclaimSecurityBasePatchSHA},
	{path: "materials.securityPatchPath", value: w001PostclaimSecurityPatchPath},
	{path: "materials.securityPatchSHA256", value: w001PostclaimSecurityPatchSHA},
	{path: "materials.hookIsolationPatchPath", value: w001PostclaimHookPatchPath},
	{path: "materials.hookIsolationPatchSHA256", value: w001PostclaimHookPatchSHA},
	{path: "materials.patchedBinarySHA256", value: w001PostclaimHookBinarySHA},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimHookFixNS},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-hook-isolation-v4.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimHookFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-signed-hook-isolation-correction-grant",
		"preserve-v3-QA-acceptance-and-Security-changes-requested-as-additive-evidence",
		"reject-config-local-selector-files-at-every-bootstrap-boundary",
		"force-workspace-hooks-disabled-for-every-helper-Beads-command",
		"deny-bootstrap-authority-through-the-hook-transaction-decorator",
		"add-local-config-and-workspace-hook-regressions",
		"correct-the-stale-public-bootstrap-status-without-rewriting-history",
		"validate-historical-v3-bytes-from-the-immutable-signed-v3-tree-and-current-bytes-from-v4",
		"edit-only-the-exact-authorized-Git-paths",
		"preserve-the-observed-canonical-M3-postimage-as-effect-evidence-not-helper-acceptance",
		"create-pinned-signer-correction-commits-and-one-signed-v4-review-tag",
		"push-the-existing-review-branch-and-v4-tag-and-rerun-the-ready-PR",
		"obtain-fresh-independent-QA-and-Security-review-before-squash-merge",
	},
	"grant.authorizedPaths": {
		w001PostclaimHookFixPath,
		w001PostclaimHookFixSig,
		"docs/evidence/W-001-validation.md",
		"docs/evidence/W-001-bootstrap-transition.md",
		"internal/authority/bootstrap/bootstrap.go",
		"internal/authority/bootstrap/bootstrap_test.go",
		w001PostclaimHookPatchPath,
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"prior-v1-through-v3-grants-signatures-tags-runs-and-history-remain-immutable",
		"v3-QA-acceptance-and-Security-changes-requested-remain-distinct-truthful-records",
		"actual-canonical-M3-postimage-remains-valid-effect-evidence-only",
		"config-local-selector-presence-fails-at-initial-fresh-effect-and-transaction-boundaries",
		"every-helper-Beads-command-forces-hooks-disabled",
		"hook-decorated-transactions-cannot-implement-bootstrap-claim-authority",
		"hook-enabled-adversarial-fixture-fails-before-read-write-or-sentinel-effect",
		"historical-and-current-integrity-bindings-are-both-required-and-neither-substitutes-for-the-other",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v4-review-tag-use-the-pinned-signer",
		"reviewed-v4-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"grant.prohibitedEffects": {
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"execute-or-replay-the-canonical-W-001-bootstrap-claim",
		"execute-any-workspace-hook-or-plugin",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"begin-gateway-runtime-platform-or-product-implementation",
		"edit-plan-manifest-feature-product-goal-or-scenario-contracts",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimPRFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimPRBindingGrant"},
	{path: "grant.id", value: "W-001-postclaim-pr-binding-v5"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T00:00:00Z"},
	{path: "grant.expiresAt", value: "2026-08-29T00:00:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimPRFixBase},
	{path: "grant.baseTree", value: w001PostclaimPRFixTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimPRFixTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimPRFixTagMsg},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "bind the active PR 8 publication vehicle after PR 7 closed without merging"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorGrant", value: "W-001-postclaim-hook-isolation-v4"},
	{path: "grant.priorGrantSHA256", value: "f0c9d5cd782350bdfefae0070b756eca1f481350cb585588a2774f06c0d6be72"},
	{path: "grant.priorGrantSignatureSHA256", value: "da013f87d13e3572ef3b2c4883ec93fdcc71f4189d6854d63aee91908d70a6f9"},
	{path: "grant.priorReviewTag", value: w001PostclaimHookFixTag},
	{path: "grant.priorReviewTagObject", value: w001PostclaimV4TagObject},
	{path: "grant.priorReviewTagTarget", value: w001PostclaimPRFixBase},
	{path: "grant.successfulRun", value: "33022606025"},
	{path: "grant.successfulJob", value: "98356474178"},
	{path: "grant.activePullRequest", value: "8"},
	{path: "grant.closedPullRequest", value: "7"},
	{path: "grant.qaDisposition", value: "changes-requested"},
	{path: "grant.securityDisposition", value: "changes-requested"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "finding.affectedPostclaimHead", value: w001PostclaimPRFixBase},
	{path: "finding.affectedPostclaimTree", value: w001PostclaimPRFixTree},
	{path: "finding.qaDisposition", value: "changes-requested"},
	{path: "finding.securityDisposition", value: "changes-requested"},
	{path: "finding.failureFingerprint", value: "stale-publication-vehicle-binding"},
	{path: "finding.failureClass", value: "foundation-owned"},
	{path: "finding.findingScope", value: "public-evidence-publication-identity"},
	{path: "finding.canonicalWorkspaceAffected", value: "false"},
	{path: "publication.repository", value: planningGrantRepository},
	{path: "publication.baseCommit", value: w001PostclaimBase},
	{path: "publication.closedPullRequest", value: "7"},
	{path: "publication.closedPullRequestMerged", value: "false"},
	{path: "publication.activePullRequest", value: "8"},
	{path: "publication.activePullRequestHead", value: w001PostclaimPRFixBase},
	{path: "publication.activePullRequestTree", value: w001PostclaimPRFixTree},
	{path: "publication.successfulRun", value: "33022606025"},
	{path: "publication.successfulJob", value: "98356474178"},
	{path: "materials.validationEvidencePath", value: "docs/evidence/W-001-validation.md"},
	{path: "materials.validationEvidenceSHA256", value: "630a4f450701efaf40a7afce59e69c6722a9551a71e755df22b5b924016702c4"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimPRFixNS},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-pr-binding-v5.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimPRFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-signed-publication-binding-grant",
		"preserve-v1-through-v4-grants-signatures-tags-runs-and-review-history",
		"record-PR-7-as-closed-unmerged-historical-evidence",
		"bind-PR-8-as-the-sole-active-publication-vehicle",
		"bind-the-successful-v4-exact-head-pull-request-run-and-job",
		"edit-only-the-exact-authorized-Git-paths",
		"create-pinned-signer-correction-commits-and-one-signed-v5-review-tag",
		"push-the-existing-review-branch-and-v5-tag-and-rerun-PR-8",
		"obtain-fresh-independent-QA-and-Security-review-before-squash-merge",
	},
	"grant.authorizedPaths": {
		w001PostclaimPRFixPath,
		w001PostclaimPRFixSig,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v4-head-tree-grant-signature-tag-and-successful-run-remain-immutable",
		"PR-7-remains-closed-and-unmerged-historical-evidence",
		"PR-8-is-the-only-current-review-and-merge-vehicle",
		"historical-v3-and-v4-material-bindings-remain-distinct-and-mandatory",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v5-review-tag-use-the-pinned-signer",
		"reviewed-v5-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"grant.prohibitedEffects": {
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"execute-or-replay-the-canonical-W-001-bootstrap-claim",
		"execute-any-workspace-hook-or-plugin",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"begin-gateway-runtime-platform-or-product-implementation",
		"edit-plan-manifest-feature-product-goal-or-scenario-contracts",
		"reopen-merge-or-mutate-PR-7",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001PostclaimChronoFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001PostclaimChronologyCorrectionGrant"},
	{path: "grant.id", value: "W-001-postclaim-chronology-correction-v6"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-26T23:47:00Z"},
	{path: "grant.expiresAt", value: "2026-08-28T23:47:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001PostclaimChronoFixBase},
	{path: "grant.baseTree", value: w001PostclaimChronoFixTree},
	{path: "grant.workingBranch", value: w001PostclaimBranch},
	{path: "grant.reviewTag", value: w001PostclaimChronoFixTag},
	{path: "grant.reviewTagMessage", value: w001PostclaimChronoFixTagMsg},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "compensate for v4 and v5 public Git effects that preceded their signed grant effective times"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.priorGrant", value: "W-001-postclaim-pr-binding-v5"},
	{path: "grant.priorGrantSHA256", value: "61b5ef56fac5916e6bd65ce884ca1080e904058688ccdc646b0f0f5502303654"},
	{path: "grant.priorGrantSignatureSHA256", value: "c040681faf0dbe2af3d231dd2735f67b12c2f166d8e137a8175cfd4ef2af362c"},
	{path: "grant.priorReviewTag", value: w001PostclaimPRFixTag},
	{path: "grant.priorReviewTagObject", value: w001PostclaimV5TagObject},
	{path: "grant.priorReviewTagTarget", value: w001PostclaimChronoFixBase},
	{path: "grant.activePullRequest", value: "8"},
	{path: "grant.qaDisposition", value: "accepted"},
	{path: "grant.securityDisposition", value: "changes-requested"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.liveLeaseAsserted", value: "false"},
	{path: "grant.implementationAllowed", value: "false"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "grant.retrospectiveAuthorizationAsserted", value: "false"},
	{path: "finding.affectedPostclaimHeads", value: "d890a96014f79438d36bde3c8967664163e9d961,765a07d3ebe2432227de7ccad65dc9f3b291deba"},
	{path: "finding.failureFingerprint", value: "grant-effective-after-governed-effects"},
	{path: "finding.failureClass", value: "foundation-owned"},
	{path: "finding.findingScope", value: "public-Git-authority-chronology"},
	{path: "finding.canonicalWorkspaceAffected", value: "false"},
	{path: "finding.retrospectiveAuthorizationAsserted", value: "false"},
	{path: "chronology.v4IssuedAt", value: "2026-08-26T23:45:00Z"},
	{path: "chronology.v4CommitAt", value: "2026-08-26T22:59:53Z"},
	{path: "chronology.v4TagAt", value: "2026-08-26T23:00:29Z"},
	{path: "chronology.v4RunStartedAt", value: "2026-08-26T23:15:21Z"},
	{path: "chronology.v5IssuedAt", value: "2026-08-27T00:00:00Z"},
	{path: "chronology.v5CommitAt", value: "2026-08-26T23:38:40Z"},
	{path: "chronology.v5TagAt", value: "2026-08-26T23:38:54Z"},
	{path: "chronology.v5RunStartedAt", value: "2026-08-26T23:40:36Z"},
	{path: "chronology.v4EffectsAuthorizedByV4", value: "false"},
	{path: "chronology.v5EffectsAuthorizedByV5", value: "false"},
	{path: "chronology.v6CompensationRequired", value: "true"},
	{path: "publication.repository", value: planningGrantRepository},
	{path: "publication.baseCommit", value: w001PostclaimBase},
	{path: "publication.closedPullRequest", value: "7"},
	{path: "publication.activePullRequest", value: "8"},
	{path: "publication.priorRun", value: "33024220053"},
	{path: "publication.priorJob", value: "98361636648"},
	{path: "materials.validationEvidencePath", value: "docs/evidence/W-001-validation.md"},
	{path: "materials.validationEvidenceSHA256", value: "8cf26301222f85537f29b3c8e70ac476c3b6819b91c19366f09b9d199e98fd13"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001PostclaimChronoFixNS},
	{path: "integrity.detachedSignature", value: "W-001-postclaim-chronology-correction-v6.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001PostclaimChronoFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"create-and-verify-this-already-effective-chronology-correction-grant",
		"preserve-v1-through-v5-grants-signatures-tags-runs-and-review-history",
		"classify-v4-and-v5-pre-effective-publication-effects-as-foundation-owned-incidents",
		"preserve-v1-through-v3-as-chronologically-valid-authority-history",
		"compensate-by-republishing-the-complete-reviewed-tree-under-v6",
		"bind-PR-8-as-the-sole-active-publication-vehicle",
		"edit-only-the-exact-authorized-Git-paths",
		"create-pinned-signer-correction-commits-and-one-signed-v6-review-tag",
		"push-the-existing-review-branch-and-v6-tag-and-rerun-PR-8",
		"obtain-fresh-independent-QA-and-Security-review-before-squash-merge",
	},
	"grant.authorizedPaths": {
		w001PostclaimChronoFixPath,
		w001PostclaimChronoFixSig,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v1-through-v5-heads-trees-grants-signatures-tags-runs-and-reviews-remain-immutable",
		"v4-and-v5-pre-effective-effects-remain-explicitly-not-retroactively-authorized",
		"v6-issuedAt-is-no-later-than-every-v6-governed-commit-and-tag-effect",
		"PR-7-remains-closed-and-unmerged-historical-evidence",
		"PR-8-is-the-only-current-review-and-merge-vehicle",
		"every-correction-commit-and-current-change-stays-inside-the-signed-path-set",
		"every-correction-commit-and-v6-review-tag-use-the-pinned-signer",
		"reviewed-v6-tree-equals-the-protected-main-squash-tree",
		"no-Beads-lease-implementation-production-or-policy-effect-is-created",
	},
	"grant.prohibitedEffects": {
		"retroactively-authorize-or-relabel-v4-or-v5-effects",
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"execute-or-replay-the-canonical-W-001-bootstrap-claim",
		"execute-any-workspace-hook-or-plugin",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"begin-gateway-runtime-platform-or-product-implementation",
		"edit-plan-manifest-feature-product-goal-or-scenario-contracts",
		"reopen-merge-or-mutate-PR-7",
		"move-delete-or-reuse-any-existing-review-tag",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
		"credentials-provider-session-customer-data-or-raw-payloads",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001DeliveryGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001DeliveryGrant"},
	{path: "grant.id", value: "W-001-delivery-v2"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T03:36:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T00:09:46Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001DeliveryBase},
	{path: "grant.baseTree", value: w001DeliveryBaseTree},
	{path: "grant.workingBranch", value: w001DeliveryBranch},
	{path: "grant.reviewTag", value: w001DeliveryReviewTag},
	{path: "grant.reviewTagMessage", value: w001DeliveryReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "republish the scanner-clean delivery tree and implement the bounded W-001 gateway"},
	{path: "grant.bead", value: "M3-W001"},
	{path: "grant.displayId", value: "W-001"},
	{path: "grant.attemptId", value: "w001-delivery-87d9680d-ca5a-4f3d-9afc-741884232e73"},
	{path: "grant.idempotencyKey", value: "w001-key"},
	{path: "grant.priorGrant", value: "W-001-postclaim-chronology-correction-v6"},
	{path: "grant.priorGrantSHA256", value: "d6789b026605a16526f8d97a9dd3c92b08c131ecaeb05564fa5b2f507ff0a445"},
	{path: "grant.priorGrantSignatureSHA256", value: "0ef51eec2a89f6d6cc298d5f725d97555ff816eeebefa72f585982df39afa982"},
	{path: "grant.priorReviewTag", value: w001PostclaimChronoFixTag},
	{path: "grant.priorReviewTagObject", value: w001PostclaimV6TagObject},
	{path: "grant.priorReviewTagTarget", value: "c6749bceb7114b16d7941afc7609c158295ccd2b"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalWorkMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "true"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.assignee", value: "work-authority-engineer"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "publication.pullRequest", value: "8"},
	{path: "publication.reviewedHead", value: "c6749bceb7114b16d7941afc7609c158295ccd2b"},
	{path: "publication.reviewedTree", value: w001DeliveryBaseTree},
	{path: "publication.mergedCommit", value: w001DeliveryBase},
	{path: "publication.mergedParent", value: w001PostclaimBase},
	{path: "publication.mergedTree", value: w001DeliveryBaseTree},
	{path: "publication.protectedMainRun", value: "33025602656"},
	{path: "publication.protectedMainJob", value: "98366054428"},
	{path: "publication.protectedMainResult", value: "SUCCESS"},
	{path: "publication.qaDisposition", value: "accepted"},
	{path: "publication.securityDisposition", value: "accepted"},
	{path: "reconciliation.commentAuthor", value: "delivery-orchestrator"},
	{path: "reconciliation.commentSHA256", value: "9c1becf8bc3e1efd7b59b41439cbb7b382f536271d5478f1750545c49fffae74"},
	{path: "reconciliation.runDisposition", value: "completed"},
	{path: "reconciliation.nextOwner", value: "work-authority-engineer"},
	{path: "reconciliation.nextNeed", value: "signed-W001-delivery-grant"},
	{path: "reconciliation.liveLeaseAsserted", value: "false"},
	{path: "reconciliation.implementationEffectObserved", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001DeliveryGrantNamespace},
	{path: "integrity.detachedSignature", value: "W-001-delivery.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001DeliveryGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"verify-the-exact-reviewed-postclaim-squash-and-protected-main-run",
		"append-one-exact-public-safe-postclaim-reconciliation-comment-to-M3-W001",
		"update-the-active-plan-and-public-evidence-with-the-verified-merge-receipt",
		"edit-only-the-exact-authorized-Git-paths",
		"implement-and-test-the-W001-authority-gateway-in-public-synthetic-fixtures",
		"create-local-non-production-databases-containers-and-network-namespaces-for-tests",
		"issue-renew-release-revoke-and-validate-only-W001-development-test-leases",
		"create-signed-semantic-commits-and-one-signed-v2-review-tag",
		"push-one-review-branch-and-tag-and-open-one-ready-pull-request",
		"obtain-independent-QA-and-Security-review-before-merge",
		"squash-merge-promptly-with-reviewed-tree-equality-and-reconcile-the-result",
	},
	"grant.authorizedPaths": {
		w001DeliveryGrantPath,
		w001DeliveryGrantSignature,
		".harness/manifest.yaml",
		canonicalActivePlan,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"internal/authority/**",
		"cmd/mars3-authority/**",
		"api/authority/**",
		"database/authority/**",
		"deploy/authority/**",
		"go.mod",
		"go.sum",
		"Makefile",
		"NOTICE",
		"THIRD_PARTY_NOTICES",
	},
	"grant.requiredProperties": {
		"Beads-Dolt-remains-the-sole-work-definition-and-lifecycle-authority",
		"PostgreSQL-remains-the-sole-live-lease-authority-and-never-becomes-ticket-authority",
		"canonical-W001-claim-and-WorkVersion-match-the-signed-preimage",
		"every-Git-write-stays-inside-the-canonical-Bead-paths-or-explicit-Orchestrator-shared-paths",
		"every-material-effect-revalidates-the-current-fencing-tuple-immediately-before-execution",
		"development-lease-generation-and-epochs-are-durable-monotonic-and-non-reusable",
		"direct-Beads-Dolt-and-lease-store-access-from-untrusted-clients-fails-closed",
		"trace-and-evidence-output-remains-bounded-public-safe-and-contains-no-raw-payloads",
		"no-canonical-lifecycle-review-or-terminal-disposition-is-created-by-the-Engineer",
		"independent-QA-and-Security-review-the-same-immutable-delivery-tree",
	},
	"grant.prohibitedEffects": {
		"replay-the-one-shot-W001-bootstrap-claim",
		"mutate-M3-W001-lifecycle-owner-dependencies-labels-metadata-or-exclusive-paths",
		"mutate-any-other-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-a-production-or-cross-ticket-lease",
		"expose-Beads-Dolt-PostgreSQL-or-provider-credentials-to-a-model-sandbox-or-public-client",
		"access-store-or-publish-secrets-private-data-provider-sessions-or-raw-payloads",
		"modify-repository-rulesets-branch-protection-secret-scanning-trust-roots-or-approval-policy",
		"production-deployment-or-destructive-migration",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001DeliveryCIFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001DeliveryCICorrection"},
	{path: "grant.id", value: "W-001-delivery-ci-correction-v3"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T08:25:00Z"},
	{path: "grant.expiresAt", value: "2026-08-28T08:25:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001DeliveryCIFixBase},
	{path: "grant.baseTree", value: w001DeliveryCIFixBaseTree},
	{path: "grant.workingBranch", value: w001DeliveryBranch},
	{path: "grant.priorGrant", value: "W-001-delivery-v2"},
	{path: "grant.priorGrantSHA256", value: "3f4d6ee6075e40ec49eefd24a9a20d734619833be61a58547d97f402258b055a"},
	{path: "grant.priorGrantSignatureSHA256", value: "3b7951067a50975875fdeb194c51680c66869868a27e94bcb53a031d4c438f45"},
	{path: "grant.priorReviewTag", value: w001DeliveryReviewTag},
	{path: "grant.priorReviewTagObject", value: w001DeliveryV2TagObject},
	{path: "grant.priorReviewTagTarget", value: w001DeliveryCIFixBase},
	{path: "grant.priorReviewTagTree", value: w001DeliveryCIFixBaseTree},
	{path: "grant.successorReviewTag", value: w001DeliveryCIFixReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001DeliveryCIFixReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "preserve the authorized v2 tag and correct the release-identity admission contract"},
	{path: "finding.code", value: "public.w001_delivery_tag_identity"},
	{path: "finding.normalizedFingerprint", value: "delivery-review-tag/release-identity-mismatch"},
	{path: "finding.observedTagger", value: "work-authority-engineer"},
	{path: "finding.requiredTagger", value: "release-manager"},
	{path: "finding.run", value: "33053544349"},
	{path: "finding.firstAttemptJob", value: "98454619462"},
	{path: "finding.retryAttemptJob", value: "98454903898"},
	{path: "finding.retryBudget", value: "exhausted"},
	{path: "finding.priorTagAuthorizedByV2", value: "true"},
	{path: "finding.priorTagAcceptedAsFinalReview", value: "false"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001DeliveryCIFixNamespace},
	{path: "integrity.detachedSignature", value: "W-001-delivery-ci-correction-v3.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001DeliveryCIFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v2-tag-head-tree-and-two-failed-check-attempts",
		"distinguish-tag-identity-failure-from-tag-target-failure",
		"validate-the-v2-engineer-tag-as-authorized-historical-evidence",
		"create-signed-correction-commits-and-one-signed-v3-release-manager-review-tag",
		"push-the-existing-review-branch-and-v3-tag-without-another-v2-run-retry",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001DeliveryCIFixPath,
		w001DeliveryCIFixSignature,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v2-tag-object-target-tree-message-signature-and-tagger-remain-immutable",
		"v2-failed-check-attempts-remain-durable-foundation-evidence",
		"v3-tag-targets-the-final-correction-head-and-uses-the-release-manager-identity",
		"pull-request-checkout-binds-the-v3-tag-to-the-event-head-not-the-synthetic-merge",
		"protected-main-tree-equals-the-signed-v3-feature-tree",
		"every-correction-commit-stays-inside-the-signed-path-set",
	},
	"grant.prohibitedEffects": {
		"move-delete-or-reuse-the-v2-review-tag",
		"retry-run-33053544349-again",
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"change-gateway-runtime-platform-or-product-behavior",
		"mutate-workflow-ruleset-repository-settings-or-secret-scanner-policy",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001DeliveryScannerFingerprints = []string{
	"0faf90716d40aa3c5251c0a9c887cc70f06cfa1e:internal/authority/beads/store_test.go:generic-api-key:96",
	"0faf90716d40aa3c5251c0a9c887cc70f06cfa1e:internal/authority/beads/store_test.go:generic-api-key:109",
	"0faf90716d40aa3c5251c0a9c887cc70f06cfa1e:internal/authority/beads/store_test.go:generic-api-key:129",
	"0faf90716d40aa3c5251c0a9c887cc70f06cfa1e:internal/authority/beads/store_test.go:generic-api-key:226",
	"2fecae29a52f0d765c8f586e9b7be3ed5ea7eeb0:.harness/grants/W-001-delivery.yaml:generic-api-key:23",
	"85848a524d40e3041199c21b89e82f2cf8910b39:internal/authority/beads/beads-v1.2.2-gateway-claim.patch:generic-api-key:283",
	"85848a524d40e3041199c21b89e82f2cf8910b39:internal/authority/beads/store_test.go:generic-api-key:196",
	"85848a524d40e3041199c21b89e82f2cf8910b39:internal/authority/beads/store_test.go:generic-api-key:365",
	"9e39ae0e7653f300635ad36f0728d5698e4eb954:internal/authority/gateway/effect_test.go:generic-api-key:217",
	"e055ddab2e1f47b1eed68593893510894a3cce7f:internal/authority/gateway/effect_test.go:generic-api-key:204",
}

var w001DeliveryScannerFixScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001DeliveryScannerCorrection"},
	{path: "grant.id", value: "W-001-delivery-scanner-correction-v4"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T11:30:30Z"},
	{path: "grant.expiresAt", value: "2026-08-28T11:30:30Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001DeliveryScannerFixBase},
	{path: "grant.baseTree", value: w001DeliveryScannerFixBaseTree},
	{path: "grant.workingBranch", value: w001DeliveryBranch},
	{path: "grant.priorGrant", value: "W-001-delivery-ci-correction-v3"},
	{path: "grant.priorGrantSHA256", value: "0448e00fbc585d95dcd740e1f8dd164882d343fbe1d73bec1cda65fd59586b85"},
	{path: "grant.priorGrantSignatureSHA256", value: "3ceeb9fed4151141c1f2f79ac2a0116698721fead88b7c4503e8b6192bd3c097"},
	{path: "grant.priorReviewTag", value: w001DeliveryCIFixReviewTag},
	{path: "grant.priorReviewTagObject", value: w001DeliveryV3TagObject},
	{path: "grant.priorReviewTagTarget", value: w001DeliveryScannerFixBase},
	{path: "grant.priorReviewTagTree", value: w001DeliveryScannerFixBaseTree},
	{path: "grant.successorReviewTag", value: w001DeliveryScannerFixReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001DeliveryScannerFixTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "preserve public delivery-v1 history while suppressing only ten immutable synthetic scanner fingerprints"},
	{path: "finding.code", value: "public.w001_delivery_history_scanner"},
	{path: "finding.normalizedFingerprint", value: "delivery-history-scanner/preserved-v1-synthetic-generic-key"},
	{path: "finding.run", value: "33066374068"},
	{path: "finding.job", value: "98497338894"},
	{path: "finding.result", value: "history-scan-failure"},
	{path: "finding.worktreeFindings", value: "0"},
	{path: "finding.historyFindings", value: "10"},
	{path: "finding.historyCommits", value: "69"},
	{path: "finding.rule", value: "generic-api-key"},
	{path: "finding.scannerImageDigest", value: "sha256:75bdb2b2f4db213cde0b8295f13a88d6b333091bbfbf3012a4e083d00d31caba"},
	{path: "finding.ignorePath", value: w001DeliveryScannerIgnorePath},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.newCanaryRequired", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001DeliveryScannerFixNamespace},
	{path: "integrity.detachedSignature", value: "W-001-delivery-scanner-correction-v4.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001DeliveryScannerFixSequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v2-and-v3-tags-heads-trees-and-failed-checks",
		"add-one-exact-ten-line-gitleaks-fingerprint-file",
		"validate-each-fingerprint-commit-path-rule-line-and-preserved-branch-ancestry",
		"prove-new-secret-canaries-remain-detectable",
		"create-signed-correction-commits-and-one-signed-v4-release-manager-review-tag",
		"push-the-existing-review-branch-and-v4-tag-without-rerunning-failed-heads",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001DeliveryScannerIgnorePath,
		w001DeliveryScannerFixPath,
		w001DeliveryScannerFixSignature,
		"docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go",
		"internal/doctrine/grant_test.go",
		"internal/doctrine/public.go",
		"internal/doctrine/doctrine_test.go",
	},
	"grant.requiredProperties": {
		"exactly-ten-full-fingerprints-and-no-pattern-wildcard-or-rule-level-exception",
		"every-fingerprint-resolves-to-the-signed-preserved-delivery-v1-history",
		"changed-extra-missing-duplicate-or-unresolvable-fingerprint-fails-closed",
		"worktree-and-new-commit-findings-remain-unsuppressed",
		"scanner-image-workflow-rules-and-canary-remain-unchanged",
		"v4-tag-targets-the-final-correction-head-and-uses-the-release-manager-identity",
		"protected-main-tree-equals-the-signed-v4-feature-tree",
	},
	"grant.prohibitedEffects": {
		"add-a-wildcard-regex-rule-path-or-commit-range-exception",
		"change-disable-or-replace-the-scanner-image-workflow-canary-or-history-scan",
		"suppress-any-future-commit-worktree-or-non-generic-api-key-finding",
		"move-delete-or-reuse-the-v2-or-v3-review-tags",
		"rerun-the-v2-or-v3-failed-heads",
		"delete-rewrite-or-force-push-the-preserved-delivery-v1-branch",
		"mutate-any-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-live-lease",
		"change-gateway-runtime-platform-or-product-behavior",
		"mutate-workflow-ruleset-repository-settings-or-trust-roots",
		"production-or-destructive-effect",
		"autonomous-mutation",
		"trust-escalation",
	},
	"finding.allowedFingerprints": w001DeliveryScannerFingerprints,
	"verification.order":          {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleGrantScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCompletionGrant"},
	{path: "grant.id", value: "W-001-lifecycle-completion-v5"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T12:05:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T12:05:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleBase},
	{path: "grant.baseTree", value: w001LifecycleBaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-delivery-scanner-correction-v4"},
	{path: "grant.priorGrantSHA256", value: "acf7b9f534b33e5a9cc6d9b439ec1d0249b7cb5116577689e37728f8e745235a"},
	{path: "grant.priorGrantSignatureSHA256", value: "afa502f4da77e04d84c531899d6324401ae32891c6ac3ca6c43130a1f9a6727b"},
	{path: "grant.priorReviewTag", value: w001DeliveryScannerFixReviewTag},
	{path: "grant.priorReviewTagObject", value: w001DeliveryV4TagObject},
	{path: "grant.priorReviewTagTarget", value: "cac4231ddcb69edd298766c5bbe3854c8269fb2a"},
	{path: "grant.priorReviewTagTree", value: w001LifecycleBaseTree},
	{path: "grant.successorReviewTag", value: w001LifecycleReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleReviewTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "complete the governed W-001 lifecycle routes before any terminal disposition"},
	{path: "grant.attemptId", value: "w001-lifecycle-completion-v5"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "true"},
	{path: "finding.code", value: "completion.w001.lifecycle_routes_missing"},
	{path: "finding.normalizedFingerprint", value: "completion-audit/governed-lifecycle-routes-missing"},
	{path: "finding.currentMain", value: w001LifecycleBase},
	{path: "finding.currentTree", value: w001LifecycleBaseTree},
	{path: "finding.pullRequest", value: "9"},
	{path: "finding.reviewedHead", value: "cac4231ddcb69edd298766c5bbe3854c8269fb2a"},
	{path: "finding.reviewedTree", value: w001LifecycleBaseTree},
	{path: "finding.protectedMainRun", value: "33069887434"},
	{path: "finding.protectedMainJob", value: "98509103754"},
	{path: "finding.qaDisposition", value: "accepted"},
	{path: "finding.securityDisposition", value: "accepted"},
	{path: "finding.missingRoutes", value: "handoff,review-verdict,run-disposition,reconciliation,terminal-transition"},
	{path: "finding.commentSHA256", value: "d7ddb1c0d4ecb00b93fcbec4d56b740da581a725e91e6381601d2d295203c38d"},
	{path: "finding.result", value: "changes-requested-same-ticket-remains-in-progress"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.assignee", value: "work-authority-engineer"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleGrantNamespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-completion-v5.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleGrantSequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v2-v3-v4-tags-heads-trees-runs-and-review-dispositions",
		"append-one-exact-public-safe-completion-audit-comment-to-M3-W001",
		"update-the-active-plan-feature-spec-evidence-and-manifest-with-the-truthful-finding",
		"implement-governed-handoff-review-run-reconciliation-and-terminal-route-contracts",
		"extend-the-pinned-native-Beads-transaction-with-bounded-lifecycle-CAS",
		"create-only-public-synthetic-development-test-leases-and-fixtures",
		"create-signed-semantic-commits-and-one-signed-v5-release-manager-review-tag",
		"push-one-review-branch-and-tag-and-open-one-ready-pull-request",
		"obtain-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleGrantPath, w001LifecycleGrantSignature, ".harness/manifest.yaml",
		canonicalActivePlan, "docs/features/F-002-work-authority.md", "docs/product-specs/work-authority.md",
		"docs/evidence/W-001-validation.md", "api/authority/v1/types.go", "internal/authority/beads/**",
		"internal/authority/gateway/**", "internal/authority/httpapi/**", "internal/authority/postgres/**",
		"database/authority/**", "cmd/mars3-authority/**", "internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"lifecycle-handoff-review-run-reconciliation-and-terminal-effects-use-typed-gateway-routes",
		"each-role-can-record-only-its-own-transition-against-one-immutable-commit",
		"changes-requested-reopens-the-same-Bead-and-never-creates-duplicate-work",
		"done-requires-QA-and-Security-acceptance-merged-SHA-completed-run-and-reconciliation",
		"every-canonical-transition-uses-one-native-Beads-CAS-and-a-monotonic-WorkVersion",
		"cross-store-handoff-and-reconciliation-unknown-outcomes-remain-recoverable-and-non-authorizing",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged-until-fresh-review-and-merge",
		"public-evidence-contains-only-bounded-hashes-identifiers-and-outcomes",
	},
	"grant.prohibitedEffects": {
		"mutate-M3-W001-lifecycle-owner-dependencies-labels-metadata-or-exclusive-paths-before-final-reconciliation",
		"mutate-any-other-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-a-production-cross-ticket-or-persistent-canonical-lease",
		"claim-create-close-supersede-or-reopen-any-canonical-Bead",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCorrectionScalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCorrectionGrant"},
	{path: "grant.id", value: "W-001-lifecycle-correction-v6"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T13:50:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T13:50:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCorrectionBase},
	{path: "grant.baseTree", value: w001LifecycleCorrectionBaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-completion-v5"},
	{path: "grant.priorGrantSHA256", value: "ee7e384447d567222975f065283d8914d4a4c1bba059888c076dbc6caac58d69"},
	{path: "grant.priorGrantSignatureSHA256", value: "e6ed28780504b9e931c2e8785b4508524ee54798eb7b9401cb265e16624bc4dd"},
	{path: "grant.priorReviewTag", value: w001LifecycleReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV5TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCorrectionBase},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCorrectionBaseTree},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.priorRun", value: "33077554760"},
	{path: "grant.priorJob", value: "98535652734"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.successorReviewTag", value: w001LifecycleCorrectionReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCorrectionTagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close the exact v5 lifecycle authority and convergence findings without canonical mutation"},
	{path: "grant.attemptId", value: "w001-lifecycle-correction-v6"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "true"},
	{path: "findings.normalizedFingerprint", value: "pr10-v5/lifecycle-authority-convergence-correction"},
	{path: "findings.reviewedHead", value: w001LifecycleCorrectionBase},
	{path: "findings.reviewedTree", value: w001LifecycleCorrectionBaseTree},
	{path: "findings.result", value: "changes-requested-same-ticket-remains-in-progress"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.assignee", value: "work-authority-engineer"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.nativeBeadsNoSkipRequired", value: "true"},
	{path: "verification.postgresNoSkipRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCorrectionNamespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-correction-v6.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCorrectionSequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v5-commits-tag-run-and-changes-requested-dispositions",
		"require-exactly-one-complete-claim-binding-for-active-and-terminal-versioned-work",
		"bind-handoff-and-replay-to-the-complete-normalized-authority-fence",
		"repair-or-reconcile-a-missing-lifecycle-receipt-before-reporting-replay-success",
		"add-truthful-blocked-and-nonterminal-disposition-details-and-append-only-recovery",
		"preserve-an-exact-reproducible-patched-Beads-build-recipe-and-hash",
		"execute-non-skipped-native-Beads-and-PostgreSQL-qualification",
		"update-the-active-plan-feature-spec-evidence-and-manifest-with-the-truthful-correction",
		"create-signed-semantic-commits-and-one-signed-v6-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-update-pull-request-10",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCorrectionPath, w001LifecycleCorrectionSignature, ".harness/manifest.yaml",
		canonicalActivePlan, "docs/features/F-002-work-authority.md", "docs/product-specs/work-authority.md",
		"docs/evidence/W-001-validation.md", "api/authority/v1/types.go", "internal/authority/beads/**",
		"internal/authority/gateway/**", "internal/authority/httpapi/**", "internal/authority/postgres/**",
		"database/authority/**", "cmd/mars3-authority/**", "internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"terminal-work-retains-exactly-one-complete-valid-claim-binding-and-detailed-evidence",
		"current-and-archived-handoffs-retain-canonical-claim-and-full-fence-digest",
		"replay-success-proves-or-repairs-the-durable-trace-receipt",
		"blocked-review-and-every-declared-noncompleted-run-have-a-recoverable-append-only-route",
		"blocker-fingerprint-next-action-and-attempt-state-remain-public-safe-and-durable",
		"completed-run-reconciliation-and-terminal-still-require-QA-and-Security-acceptance",
		"native-and-PostgreSQL-qualification-execute-without-skips-and-reproduce-bound-hashes",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged-until-fresh-review-and-merge",
	},
	"grant.prohibitedEffects": {
		"mutate-M3-W001-lifecycle-owner-dependencies-labels-metadata-or-exclusive-paths",
		"mutate-any-other-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"move-delete-or-reuse-the-v5-review-tag",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"lifecycle.terminal_claim_binding_fail_open",
		"lifecycle.handoff_replay_fence_splice",
		"lifecycle.missing_receipt_replay_success",
		"lifecycle.nonterminal_convergence_deadlock",
		"lifecycle.qualification_not_reproducible",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCorrectionV7Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCorrectionGrant"},
	{path: "grant.id", value: "W-001-lifecycle-correction-v7"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T15:07:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T15:07:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCorrectionV7Base},
	{path: "grant.baseTree", value: w001LifecycleCorrectionV7BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-correction-v6"},
	{path: "grant.priorGrantSHA256", value: "0d5b8aa829d8eecd4755f6063512facdcd17c0312e45ec4ca2ff1295f73a462e"},
	{path: "grant.priorGrantSignatureSHA256", value: "cae00cafd4007330beb5bac423300cee4b7dfc349d7e5385c57f62092a9a5a8d"},
	{path: "grant.priorReviewTag", value: w001LifecycleCorrectionReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV6TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCorrectionV7Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCorrectionV7BaseTree},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.priorRun", value: "33083662143"},
	{path: "grant.priorJob", value: "98557343299"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.successorReviewTag", value: w001LifecycleCorrectionV7ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCorrectionV7TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close the exact v6 claim-lineage convergence and reproducible-build findings without canonical mutation"},
	{path: "grant.attemptId", value: "w001-lifecycle-correction-v7"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "true"},
	{path: "findings.normalizedFingerprint", value: "pr10-v6/lifecycle-lineage-retry-build-correction"},
	{path: "findings.reviewedHead", value: w001LifecycleCorrectionV7Base},
	{path: "findings.reviewedTree", value: w001LifecycleCorrectionV7BaseTree},
	{path: "findings.result", value: "changes-requested-same-ticket-remains-in-progress"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.assignee", value: "work-authority-engineer"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.nativeBeadsNoSkipRequired", value: "true"},
	{path: "verification.postgresNoSkipRequired", value: "true"},
	{path: "verification.independentColdBuildsRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCorrectionV7Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-correction-v7.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCorrectionV7Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v6-commits-tag-run-and-changes-requested-dispositions",
		"join-every-current-and-archived-handoff-to-the-sole-complete-canonical-claim",
		"reject-null-malformed-incomplete-or-type-confused-claim-shadows-and-contradictory-legacy-lifecycle-scalars",
		"require-a-normalized-fingerprint-for-every-noncompleted-run",
		"enforce-one-retry-per-fingerprint-and-durable-blocked-escalation-without-a-third-automatic-attempt",
		"preserve-a-hermetic-independent-reproducible-patched-Beads-build-recipe-source-digest-and-binary-hash",
		"execute-non-skipped-native-Beads-and-PostgreSQL-qualification",
		"update-the-active-plan-feature-spec-evidence-and-manifest-with-the-truthful-v7-correction",
		"create-signed-semantic-commits-and-one-signed-v7-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-update-pull-request-10",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCorrectionV7Path, w001LifecycleCorrectionV7Signature, ".harness/manifest.yaml",
		canonicalActivePlan, "docs/features/F-002-work-authority.md", "docs/product-specs/work-authority.md",
		"docs/evidence/W-001-validation.md", "api/authority/v1/types.go", "internal/authority/beads/**",
		"internal/authority/gateway/**", "internal/authority/httpapi/**", "internal/authority/postgres/**",
		"database/authority/**", "cmd/mars3-authority/**", "internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"every-current-and-archived-handoff-canonical-attempt-equals-the-sole-retained-claim-attempt",
		"work-and-bootstrap-claims-have-distinct-complete-type-specific-contracts-and-raw-presence-is-fail-closed",
		"legacy-lifecycle-scalars-are-absent-or-exactly-derived-from-detailed-records",
		"every-noncompleted-run-retains-a-public-safe-normalized-fingerprint",
		"first-failure-and-one-retry-use-monotonic-attempts-and-equivalent-recurrence-becomes-blocked",
		"a-third-equivalent-automatic-attempt-is-denied-across-current-and-archived-cycles",
		"independent-cold-builds-from-different-source-and-cache-paths-reproduce-the-bound-binary",
		"native-and-PostgreSQL-qualification-execute-without-skips",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged-until-fresh-review-and-merge",
	},
	"grant.prohibitedEffects": {
		"mutate-M3-W001-lifecycle-owner-dependencies-labels-metadata-or-exclusive-paths",
		"mutate-any-other-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"move-delete-or-reuse-the-v6-review-tag",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"lifecycle.claim_lineage_not_joined",
		"lifecycle.failure_fingerprint_retry_not_monotonic",
		"lifecycle.qualification_not_independently_reproducible",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCorrectionV8Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCorrectionGrant"},
	{path: "grant.id", value: "W-001-lifecycle-correction-v8"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T16:29:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T16:29:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCorrectionV8Base},
	{path: "grant.baseTree", value: w001LifecycleCorrectionV8BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-correction-v7"},
	{path: "grant.priorGrantSHA256", value: "ee5866add69534ff82dafba945d17959949584f91746551c67be3af70a9fbff2"},
	{path: "grant.priorGrantSignatureSHA256", value: "33070c3d15d4e9b121c3812538a47510a8bd3b996ebb845c94ebb648e3148009"},
	{path: "grant.priorReviewTag", value: w001LifecycleCorrectionV7ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV7TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCorrectionV8Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCorrectionV8BaseTree},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.priorRun", value: "33091727157"},
	{path: "grant.priorJob", value: "98586729802"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.successorReviewTag", value: w001LifecycleCorrectionV8ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCorrectionV8TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close the exact v7 canonical-key legacy-scalar and dependency-readiness findings without canonical mutation"},
	{path: "grant.attemptId", value: "w001-lifecycle-correction-v8"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "true"},
	{path: "findings.normalizedFingerprint", value: "pr10-v7/canonical-key-legacy-dependency-correction"},
	{path: "findings.reviewedHead", value: w001LifecycleCorrectionV8Base},
	{path: "findings.reviewedTree", value: w001LifecycleCorrectionV8BaseTree},
	{path: "findings.result", value: "changes-requested-same-ticket-remains-in-progress"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.assignee", value: "work-authority-engineer"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.nativeBeadsNoSkipRequired", value: "true"},
	{path: "verification.postgresNoSkipRequired", value: "true"},
	{path: "verification.independentColdBuildsRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCorrectionV8Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-correction-v8.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCorrectionV8Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v7-commits-tag-run-and-changes-requested-dispositions",
		"reject-case-folded-or-other-noncanonical-authority-metadata-keys-before-typed-decoding",
		"require-legacy-lifecycle-scalars-to-be-empty-when-detailed-evidence-is-absent",
		"derive-dependency-readiness-from-consistent-detailed-evidence-or-reject-the-projection",
		"mirror-the-corrections-in-the-pinned-native-Beads-transaction-path",
		"execute-non-skipped-native-Beads-and-PostgreSQL-qualification",
		"preserve-an-independent-reproducible-patched-Beads-source-digest-and-binary-hash",
		"update-the-active-plan-feature-spec-evidence-and-manifest-with-the-truthful-v8-correction",
		"create-signed-semantic-commits-and-one-signed-v8-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-update-pull-request-10",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCorrectionV8Path, w001LifecycleCorrectionV8Signature, ".harness/manifest.yaml",
		canonicalActivePlan, "docs/features/F-002-work-authority.md", "docs/product-specs/work-authority.md",
		"docs/evidence/W-001-validation.md", "internal/authority/beads/**",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"authority-metadata-object-keys-are-unique-and-canonical-under-case-folding-before-typed-decoding",
		"active-records-without-detailed-evidence-cannot-claim-accepted-completed-reconciled-or-blocked-state",
		"detailed-dependency-lifecycle-records-and-legacy-scalars-cannot-contradict-each-other",
		"dependency-readiness-is-derived-from-valid-detailed-evidence-when-it-exists",
		"project-adapter-and-native-transaction-enforce-the-same-fail-closed-contract",
		"native-and-PostgreSQL-qualification-execute-without-skips",
		"independent-cold-builds-reproduce-the-bound-patched-Beads-binary",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged-until-fresh-review-and-merge",
	},
	"grant.prohibitedEffects": {
		"mutate-M3-W001-lifecycle-owner-dependencies-labels-metadata-or-exclusive-paths",
		"mutate-any-other-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"move-delete-or-reuse-the-v7-review-tag",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"lifecycle.claim_key_alias_not_canonical",
		"lifecycle.legacy_active_scalar_contradiction",
		"lifecycle.dependency_detailed_state_ignored",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCorrectionV9Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCorrectionGrant"},
	{path: "grant.id", value: "W-001-lifecycle-correction-v9"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T17:29:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T17:29:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCorrectionV9Base},
	{path: "grant.baseTree", value: w001LifecycleCorrectionV9BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-correction-v8"},
	{path: "grant.priorGrantSHA256", value: "bfd0f0e623d5640eeb1f1dce7d2d88fee5034052e73e446979ec7dfdd238a699"},
	{path: "grant.priorGrantSignatureSHA256", value: "e55db7343aad3d2f81a4626e082501dbf9bc693056222b9514ea378456ec7296"},
	{path: "grant.priorReviewTag", value: w001LifecycleCorrectionV8ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV8TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCorrectionV9Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCorrectionV9BaseTree},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.priorRun", value: "33097381660"},
	{path: "grant.priorJob", value: "98606506699"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.successorReviewTag", value: w001LifecycleCorrectionV9ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCorrectionV9TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "work-authority-engineer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close the exact v8 recursive-native-key and dependency-lineage-stripping findings without canonical mutation"},
	{path: "grant.attemptId", value: "w001-lifecycle-correction-v9"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "true"},
	{path: "findings.normalizedFingerprint", value: "pr10-v8/native-recursive-key-dependency-lineage-correction"},
	{path: "findings.reviewedHead", value: w001LifecycleCorrectionV9Base},
	{path: "findings.reviewedTree", value: w001LifecycleCorrectionV9BaseTree},
	{path: "findings.result", value: "changes-requested-same-ticket-remains-in-progress"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.assignee", value: "work-authority-engineer"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.nativeBeadsNoSkipRequired", value: "true"},
	{path: "verification.postgresNoSkipRequired", value: "true"},
	{path: "verification.independentColdBuildsRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCorrectionV9Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-correction-v9.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCorrectionV9Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v8-commits-tag-run-and-changes-requested-dispositions",
		"enforce-recursive-exact-schema-keys-at-every-native-authority-metadata-object-boundary",
		"deny-sparse-legacy-readiness-for-versioned-claim-bearing-or-detailed-key-bearing-dependencies",
		"execute-the-full-recursive-native-key-and-dependency-lineage-stripping-regression-corpus",
		"execute-non-skipped-native-Beads-and-PostgreSQL-qualification",
		"preserve-an-independent-reproducible-patched-Beads-source-digest-and-binary-hash",
		"update-the-active-plan-feature-spec-evidence-and-manifest-with-the-truthful-v9-correction",
		"create-signed-semantic-commits-and-one-signed-v9-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-update-pull-request-10",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCorrectionV9Path, w001LifecycleCorrectionV9Signature, ".harness/manifest.yaml",
		canonicalActivePlan, "docs/features/F-002-work-authority.md", "docs/product-specs/work-authority.md",
		"docs/evidence/W-001-validation.md", "internal/authority/beads/**",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"native-authority-metadata-object-keys-are-unique-and-canonical-recursively-before-semantic-parsing",
		"recursive-native-key-validation-covers-work-version-claims-handoffs-reviews-runs-failures-reconciliation-terminal-and-history",
		"legacy-dependency-readiness-is-limited-to-an-explicitly-unversioned-unclaimed-and-detail-key-absent-shape",
		"versioned-claim-bearing-or-detailed-key-bearing-dependencies-require-the-complete-valid-terminal-chain",
		"project-adapter-and-native-transaction-enforce-the-same-fail-closed-contract",
		"native-and-PostgreSQL-qualification-execute-without-skips",
		"independent-cold-builds-reproduce-the-bound-patched-Beads-binary",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged-until-fresh-review-and-merge",
	},
	"grant.prohibitedEffects": {
		"mutate-M3-W001-lifecycle-owner-dependencies-labels-metadata-or-exclusive-paths",
		"mutate-any-other-Bead-Dolt-row-dependency-label-comment-or-history",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"move-delete-or-reuse-the-v8-review-tag",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"lifecycle.native_recursive_key_alias_not_canonical",
		"lifecycle.dependency_lineage_stripping",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleStabilizationV10Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIStabilizationGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-stabilization-v10"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T18:43:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T18:43:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleStabilizationV10Base},
	{path: "grant.baseTree", value: w001LifecycleStabilizationV10BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-correction-v9"},
	{path: "grant.priorReviewTag", value: w001LifecycleCorrectionV9ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV9TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleStabilizationV10Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleStabilizationV10BaseTree},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.failedRun", value: "33104553091"},
	{path: "grant.failedAttemptOneJob", value: "98630789458"},
	{path: "grant.failedAttemptTwoJob", value: "98631170195"},
	{path: "grant.priorQADisposition", value: "not-requested-ci-blocked"},
	{path: "grant.priorSecurityDisposition", value: "not-requested-ci-blocked"},
	{path: "grant.successorReviewTag", value: w001LifecycleStabilizationV10ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleStabilizationV10TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "stabilize disposable doctrine Git fixtures after two bounded identical pack-cleanup races without changing authority behavior"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-stabilization-v10"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "failure.normalizedFingerprint", value: "ci/doctrine-tempdir-git-pack-cleanup"},
	{path: "failure.outcome", value: "exhausted-two-identical-retries"},
	{path: "failure.attemptOneFinding", value: "TestW001LifecycleCorrectionV7GrantFailsClosed TempDir RemoveAll pack directory not empty"},
	{path: "failure.attemptTwoFinding", value: "TestW001LifecycleCompletionGrantFailsClosed and TestW001LifecycleCorrectionGrantFailsClosed TempDir RemoveAll pack directory not empty"},
	{path: "failure.nextAction", value: "prospective-test-fixture-stabilization"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleStabilizationV10Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-stabilization-v10.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleStabilizationV10Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-commit-tree-tag-and-two-failed-CI-attempts",
		"record-the-normalized-TempDir-Git-pack-cleanup-fingerprint-and-exhausted-retry",
		"disable-auto-maintenance-auto-gc-and-detached-background-cleanup-on-every-disposable-test-Git-command",
		"assert-the-test-only-Git-wrapper-applies-the-bounded-configuration",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-stabilization",
		"create-signed-semantic-commits-and-one-signed-v10-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-rerun-pull-request-10-once",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleStabilizationV10Path, w001LifecycleStabilizationV10Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-and-qualification-bytes-remain-unchanged",
		"every-disposable-test-Git-command-disables-maintenance-auto-gc-auto-detach-and-maintenance-auto-detach",
		"the-correction-mutates-no-user-global-system-or-production-Git-configuration",
		"both-v9-CI-attempts-remain-preserved-as-foundation-owned-failures",
		"the-next-public-run-uses-the-exact-signed-v10-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-or-API-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-the-failed-v9-commit-a-third-time",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"move-delete-or-reuse-the-v9-review-tag",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIFencingV11Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIFencingCorrectionGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-fencing-v11"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T19:05:55Z"},
	{path: "grant.expiresAt", value: "2026-08-30T19:05:55Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIFencingV11Base},
	{path: "grant.baseTree", value: w001LifecycleCIFencingV11BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-stabilization-v10"},
	{path: "grant.priorGrantSHA256", value: "b6f29734dabbeaff52f96c8d5e0a8910fadf81c841f4d9fd4a7cd799add586f9"},
	{path: "grant.priorGrantSignatureSHA256", value: "9bc96e8c5a0f35fee3998066bd6fa00b45b671ee1b66debad4f6e21c7341ab32"},
	{path: "grant.priorReviewTag", value: w001LifecycleStabilizationV10ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV10TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIFencingV11Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIFencingV11BaseTree},
	{path: "grant.priorRun", value: "33105792480"},
	{path: "grant.priorJob", value: "98635155160"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIFencingV11ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIFencingV11TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "route every disposable test Git operation through one sanitized command-local wrapper without persistent repository configuration"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-fencing-v11"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.rawCloneFinding", value: "grant_test.go raw git clone bypassed the bounded wrapper and inherited global configuration"},
	{path: "findings.persistedConfigFinding", value: "disposable fixtures wrote maintenance.auto and gc.auto into repository-local configuration"},
	{path: "findings.nextAction", value: "prospective-command-local-test-Git-fencing-correction"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIFencingV11Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-fencing-v11.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIFencingV11Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-commit-tree-tag-run-and-dispositions",
		"route-pre-repository-clone-and-every-other-disposable-test-Git-operation-through-the-bounded-wrapper",
		"remove-disposable-repository-persistent-maintenance-and-gc-configuration",
		"assert-effective-command-local-fences-and-absent-local-and-global-persistent-values",
		"reject-any-raw-disposable-Git-command-outside-the-single-audited-wrapper",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-correction",
		"create-signed-semantic-commits-and-one-signed-v11-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIFencingV11Path, w001LifecycleCIFencingV11Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"every-disposable-test-Git-operation-including-pre-repository-clone-uses-the-sanitized-bounded-wrapper",
		"effective-maintenance-auto-gc-auto-detach-and-maintenance-auto-detach-remain-command-local-and-disabled",
		"no-disposable-repository-local-global-user-system-or-production-Git-configuration-is-mutated",
		"v10-QA-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v11-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-or-v10-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes":     {"ci.test_git_sanitization_incomplete", "ci.test_git_configuration_persisted"},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIHardeningV12Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIHardeningGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-hardening-v12"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T19:32:00Z"},
	{path: "grant.expiresAt", value: "2026-08-30T19:32:00Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIHardeningV12Base},
	{path: "grant.baseTree", value: w001LifecycleCIHardeningV12BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-fencing-v11"},
	{path: "grant.priorGrantSHA256", value: "6df1dc4978e6b3657986ef43a41aaa3437567772c95bec4f151d8abcf0e9396b"},
	{path: "grant.priorGrantSignatureSHA256", value: "967efc83af964fc5abaf42f28cad1ad0231dc35524eecf5f9a05eae093d80b0e"},
	{path: "grant.priorReviewTag", value: w001LifecycleCIFencingV11ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV11TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIHardeningV12Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIHardeningV12BaseTree},
	{path: "grant.priorRun", value: "33108126981"},
	{path: "grant.priorJob", value: "98643418071"},
	{path: "grant.priorQADisposition", value: "accepted"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIHardeningV12ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIHardeningV12TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "make disposable test Git execution non-overridable and close ambient process-execution paths"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-hardening-v12"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.callerOverrideFinding", value: "later caller configuration could override every bounded maintenance fence"},
	{path: "findings.environmentInjectionFinding", value: "inherited Git exec and template paths could run hostile upload-pack and hook executables"},
	{path: "findings.guardFinding", value: "the literal one-file guard missed aliases concatenation CommandContext shells and alternate files"},
	{path: "findings.nextAction", value: "prospective-test-process-and-Git-execution-hardening"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIHardeningV12Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-hardening-v12.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIHardeningV12Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-v11-immutable-history-runs-and-dispositions",
		"pin-disposable-test-Git-execution-to-the-trusted-absolute-binary",
		"strip-all-ambient-Git-environment-variables-before-reintroducing-the-exact-bounded-set",
		"reject-protected-c-config-env-template-upload-pack-exec-path-and-persistent-config-mutations",
		"replace-the-one-file-literal-guard-with-repository-wide-AST-process-invocation-admission",
		"add-adversarial-caller-environment-alias-concatenation-command-context-shell-and-alternate-file-regressions",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-hardening",
		"create-signed-semantic-commits-and-one-signed-v12-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIHardeningV12Path, w001LifecycleCIHardeningV12Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"caller-arguments-cannot-override-any-of-the-four-command-local-fences",
		"inherited-Git-exec-template-hook-config-and-repository-redirection-state-is-unavailable",
		"protected-local-global-system-configuration-mutations-and-hostile-clone-options-fail-before-Git-executes",
		"every-test-process-invocation-is-AST-admitted-across-all-doctrine-test-files-and-aliases",
		"only-two-exact-ssh-keygen-test-calls-and-the-one-trusted-Git-wrapper-call-are-admitted",
		"v11-QA-accepted-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v12-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-v10-or-v11-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes":     {"ci.test_git_fences_caller_overridable", "ci.test_git_environment_execution_injection", "ci.test_process_guard_fail_open"},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIHardeningV13Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIHardeningGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-hardening-v13"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T20:00:30Z"},
	{path: "grant.expiresAt", value: "2026-08-30T20:00:30Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIHardeningV13Base},
	{path: "grant.baseTree", value: w001LifecycleCIHardeningV13BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-hardening-v12"},
	{path: "grant.priorGrantSHA256", value: "9356f21a72ce652b6238be15f6393cf402c34408fe467526e43f2a53c7ca5ab1"},
	{path: "grant.priorGrantSignatureSHA256", value: "733e67f5ec4807523ab56febc60c21aed3f4a2cfe58253febc58cef3b5d113d1"},
	{path: "grant.priorReviewTag", value: w001LifecycleCIHardeningV12ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV12TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIHardeningV13Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIHardeningV13BaseTree},
	{path: "grant.priorRun", value: "33110339883"},
	{path: "grant.priorJob", value: "98651204635"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIHardeningV13ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIHardeningV13TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "replace residual Git argument and process invocation denylists with closed admission"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-hardening-v13"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.gitFinding", value: "abbreviated compact and config-producing Git arguments escaped the denylist"},
	{path: "findings.processFinding", value: "direct exec Cmd indirect syscall function values and nested tests escaped process admission"},
	{path: "findings.nextAction", value: "prospective-closed-argv-and-recursive-process-admission"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIHardeningV13Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-hardening-v13.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIHardeningV13Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-v12-immutable-history-runs-and-dispositions",
		"replace-disposable-test-Git-option-denylists-with-exact-per-subcommand-argv-schemas",
		"reject-long-option-abbreviations-compact-options-outside-root-metadata-and-config-producing-subcommands",
		"recursively-enumerate-the-doctrine-test-tree-for-process-invocation-admission",
		"reject-direct-exec-Cmd-construction-and-indirect-os-or-syscall-process-function-values",
		"retain-and-extend-the-hostile-environment-alias-concatenation-command-context-shell-and-persistence-regressions",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-hardening",
		"create-signed-semantic-commits-and-one-signed-v13-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIHardeningV13Path, w001LifecycleCIHardeningV13Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"every-disposable-test-Git-command-matches-one-exact-subcommand-and-argv-schema",
		"Git-long-option-abbreviations-compact-upload-pack-and-outside-root-or-config-effects-are-unrepresentable",
		"doctrine-test-process-enumeration-is-recursive-and-fails-closed-on-every-unreadable-or-unparseable-file",
		"direct-exec-Cmd-and-indirect-os-or-syscall-process-construction-or-invocation-are-denied",
		"only-two-exact-ssh-keygen-test-calls-and-the-one-trusted-Git-wrapper-call-are-admitted",
		"v12-QA-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v13-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-v10-v11-or-v12-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes":     {"ci.test_git_argv_schema_fail_open", "ci.test_process_guard_incomplete"},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIHardeningV14Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIHardeningGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-hardening-v14"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-27T21:00:28Z"},
	{path: "grant.expiresAt", value: "2026-08-30T21:00:28Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIHardeningV14Base},
	{path: "grant.baseTree", value: w001LifecycleCIHardeningV14BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-hardening-v13"},
	{path: "grant.priorGrantSHA256", value: "221545ade5766928436cf75c30250c32780305f5e7d12f8a60bb4d35659d109d"},
	{path: "grant.priorGrantSignatureSHA256", value: "0984d7e54f0f93e3514c0ecb3cdd0cd5716d05a7b4571068648b47abcadf056f"},
	{path: "grant.priorReviewTag", value: w001LifecycleCIHardeningV13ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV13TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIHardeningV14Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIHardeningV14BaseTree},
	{path: "grant.priorRun", value: "33112938711"},
	{path: "grant.priorJob", value: "98660186954"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIHardeningV14ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIHardeningV14TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close physical clone containment and test-to-production Git executor bypasses"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-hardening-v14"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.cloneFinding", value: "lexical containment admitted a symlinked clone ancestor that redirected writes outside the disposable root"},
	{path: "findings.processFinding", value: "doctrine tests called the production Git executor directly and bypassed test wrapper admission"},
	{path: "findings.nextAction", value: "prospective-physical-clone-and-test-Git-executor-isolation"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIHardeningV14Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-hardening-v14.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIHardeningV14Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-v13-immutable-history-runs-and-dispositions",
		"require-a-direct-canonical-disposable-root-and-reject-symlinked-clone-target-ancestors-or-existing-targets",
		"resolve-the-existing-clone-parent-and-prove-physical-containment-before-command-construction",
		"route-the-three-existing-test-read-calls-through-the-exact-test-Git-wrapper",
		"extend-only-the-exact-read-only-cat-file-and-show-argv-schemas-needed-by-those-calls",
		"reject-test-calls-to-the-production-planning-Grant-Git-executor-in-every-doctrine-test-file",
		"retain-and-extend-the-recursive-process-and-physical-path-adversarial-regressions",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-hardening",
		"create-signed-semantic-commits-and-one-signed-v14-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIHardeningV14Path, w001LifecycleCIHardeningV14Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"clone-target-admission-uses-canonical-physical-ancestry-and-rejects-symlinked-roots-ancestors-and-targets",
		"every-admitted-clone-target-is-a-nonexistent-direct-child-of-the-canonical-disposable-root",
		"all-test-authored-Git-processes-route-through-the-exact-test-wrapper-and-closed-argv-schemas",
		"no-doctrine-test-file-calls-the-production-planning-Grant-Git-executor-directly",
		"process-and-path-enumeration-fails-closed-on-unreadable-unparseable-or-symlinked-surfaces",
		"v13-QA-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v14-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-v10-v11-v12-or-v13-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes":     {"ci.test_git_clone_physical_escape", "ci.test_process_guard_transitive_bypass"},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIHardeningV15Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIHardeningGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-hardening-v15"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-28T08:32:45Z"},
	{path: "grant.expiresAt", value: "2026-08-31T08:32:45Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIHardeningV15Base},
	{path: "grant.baseTree", value: w001LifecycleCIHardeningV15BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-hardening-v14"},
	{path: "grant.priorGrantSHA256", value: "a9dd38a11cf0f076b8af79618739a303e69a63b41caca1c2fb23f8ada87e3eac"},
	{path: "grant.priorGrantSignatureSHA256", value: "98d439533d5754ea7e52c66de0ff3ffe3348daf1aa6754038f16c68f2493cded"},
	{path: "grant.priorReviewTag", value: w001LifecycleCIHardeningV14ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV14TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIHardeningV15Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIHardeningV15BaseTree},
	{path: "grant.priorRun", value: "33123061855"},
	{path: "grant.priorJob", value: "98694494697"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIHardeningV15ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIHardeningV15TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close test Git physical binding races and dot-import process admission bypasses"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-hardening-v15"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.pathFinding", value: "a real disposable root reached through a symlinked ancestor was admitted as canonical"},
	{path: "findings.raceFinding", value: "clone target reservation was released before Git consumed the writable pathname"},
	{path: "findings.processFinding", value: "dot-imported os and syscall process entrypoints escaped selector-only admission"},
	{path: "findings.nextAction", value: "prospective-descriptor-bound-test-Git-and-closed-process-import-admission"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIHardeningV15Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-hardening-v15.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIHardeningV15Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-v14-immutable-history-runs-and-dispositions",
		"remove-the-writable-Git-clone-subprocess-from-doctrine-test-fixtures-and-its-argv-schema",
		"require-every-test-Git-root-to-be-one-direct-canonical-directory-without-symlinked-ancestry",
		"bind-every-test-Git-process-to-the-verified-open-directory-descriptor-through-execution",
		"initialize-fetch-and-checkout-only-through-the-existing-closed-test-Git-wrapper",
		"reject-dot-or-blank-imports-of-os-os-exec-and-syscall-across-the-recursive-doctrine-test-surface",
		"retain-and-extend-all-v10-v14-environment-argv-process-and-physical-path-regressions",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-hardening",
		"reproduce-the-pinned-v9-native-Beads-artifact-and-non-skipped-conformance-without-changing-patch-bytes",
		"create-signed-semantic-commits-and-one-signed-v15-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIHardeningV15Path, w001LifecycleCIHardeningV15Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"no-doctrine-test-Git-command-admits-or-executes-the-clone-subcommand",
		"every-test-Git-root-is-lexically-and-physically-identical-and-held-open-through-child-execution",
		"root-or-ancestor-replacement-cannot-redirect-any-admitted-test-Git-process",
		"all-test-authored-Git-processes-route-through-the-exact-wrapper-and-closed-argv-schemas",
		"dot-and-blank-os-os-exec-or-syscall-imports-fail-closed-before-process-call-analysis",
		"process-and-path-enumeration-fails-closed-on-unreadable-unparseable-or-symlinked-surfaces",
		"v14-QA-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v15-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-v10-v11-v12-v13-or-v14-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"ci.test_git_root_ancestor_alias_admitted",
		"ci.test_git_clone_reservation_toctou",
		"ci.test_process_guard_dot_import_bypass",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIHardeningV16Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIHardeningGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-hardening-v16"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-28T11:11:48Z"},
	{path: "grant.expiresAt", value: "2026-08-31T11:11:48Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIHardeningV16Base},
	{path: "grant.baseTree", value: w001LifecycleCIHardeningV16BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-hardening-v15"},
	{path: "grant.priorGrantSHA256", value: "b751be17403e0b41f75d853e0d8a4e6baa61101436a94bf61c47b42366ac409b"},
	{path: "grant.priorGrantSignatureSHA256", value: "de2e0c92114e3aa4a4639206caa8a97d1ec39e8ab819989c7335c0df107e458e"},
	{path: "grant.priorReviewTag", value: w001LifecycleCIHardeningV15ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV15TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIHardeningV16Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIHardeningV16BaseTree},
	{path: "grant.priorRun", value: "33165311496"},
	{path: "grant.priorJob", value: "98829194619"},
	{path: "grant.priorQADisposition", value: "changes-requested"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIHardeningV16ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIHardeningV16TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close descriptor-helper provenance and executable-image substitution bypasses"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-hardening-v16"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.provenanceFinding", value: "a same-package test could directly call the allowlisted helper with ambient environment and descriptor state"},
	{path: "findings.imageFinding", value: "the mutable test-executable pathname was resolved before a deterministic substitution hook and executed without image binding"},
	{path: "findings.nextAction", value: "prospective-fixed-descriptor-trampoline-and-closed-test-process-call-graph"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIHardeningV16Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-hardening-v16.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIHardeningV16Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-v15-immutable-history-runs-and-dispositions",
		"remove-the-self-executable-descriptor-helper-TestMain-mode-and-environment-argument-channel",
		"launch-only-the-exact-system-Perl-descriptor-trampoline-with-fixed-non-input-code",
		"have-that-trampoline-fchdir-only-to-inherited-root-descriptor-three-and-exec-only-literal-usr-bin-git",
		"reject-dynamic-test-process-executables-and-direct-or-transitive-calls-to-removed-helper-surfaces",
		"restrict-test-Git-fetch-sources-to-canonical-local-directories-and-the-file-protocol",
		"retain-and-extend-all-v10-v15-environment-argv-process-and-physical-path-regressions",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-hardening",
		"reproduce-the-pinned-v9-native-Beads-artifact-and-non-skipped-conformance-without-changing-patch-bytes",
		"create-signed-semantic-commits-and-one-signed-v16-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIHardeningV16Path, w001LifecycleCIHardeningV16Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"no-test-process-executes-the-Go-test-image-or-selects-helper-mode-from-the-environment",
		"the-only-test-Git-process-constructor-uses-one-literal-system-executable-and-one-byte-exact-trampoline",
		"the-trampoline-consumes-no-code-or-command-from-input-and-enters-only-inherited-root-descriptor-three",
		"root-or-ancestor-replacement-cannot-redirect-any-admitted-test-Git-process",
		"direct-helper-calls-dynamic-executables-and-unexpected-process-callers-fail-closed-under-recursive-admission",
		"fetch-accepts-only-canonical-local-directories-with-network-protocols-disabled",
		"process-and-path-enumeration-fails-closed-on-unreadable-unparseable-or-symlinked-surfaces",
		"v15-QA-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v16-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-v10-v11-v12-v13-v14-or-v15-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"ci.test_git_descriptor_helper_transitive_bypass",
		"ci.test_git_helper_executable_path_toctou",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

var w001LifecycleCIHardeningV17Scalars = []grantScalarExpectation{
	{path: "schemaVersion", value: "1"},
	{path: "kind", value: "MARS3W001LifecycleCIHardeningGrant"},
	{path: "grant.id", value: "W-001-lifecycle-ci-hardening-v17"},
	{path: "grant.classification", value: "PUBLIC"},
	{path: "grant.issuedAt", value: "2026-08-28T20:33:09Z"},
	{path: "grant.expiresAt", value: "2026-08-31T20:33:09Z"},
	{path: "grant.repository", value: planningGrantRepository},
	{path: "grant.baseCommit", value: w001LifecycleCIHardeningV17Base},
	{path: "grant.baseTree", value: w001LifecycleCIHardeningV17BaseTree},
	{path: "grant.workingBranch", value: w001LifecycleBranch},
	{path: "grant.priorGrant", value: "W-001-lifecycle-ci-hardening-v16"},
	{path: "grant.priorGrantSHA256", value: "95fa2caa2befd270ed15f9c317a37ceec442b70c0826a869323d34c3d612d835"},
	{path: "grant.priorGrantSignatureSHA256", value: "26711612175f7969ec168159e535e7b6b7273641690ed4c486e609cddaa844e5"},
	{path: "grant.priorReviewTag", value: w001LifecycleCIHardeningV16ReviewTag},
	{path: "grant.priorReviewTagObject", value: w001LifecycleV16TagObject},
	{path: "grant.priorReviewTagTarget", value: w001LifecycleCIHardeningV17Base},
	{path: "grant.priorReviewTagTree", value: w001LifecycleCIHardeningV17BaseTree},
	{path: "grant.priorRun", value: "33206197037"},
	{path: "grant.priorJob", value: "98967743138"},
	{path: "grant.priorQADisposition", value: "accepted"},
	{path: "grant.priorSecurityDisposition", value: "changes-requested"},
	{path: "grant.pullRequest", value: "10"},
	{path: "grant.successorReviewTag", value: w001LifecycleCIHardeningV17ReviewTag},
	{path: "grant.successorReviewTagMessage", value: w001LifecycleCIHardeningV17TagMessage},
	{path: "grant.signerRole", value: "human-bootstrap-authority"},
	{path: "grant.coordinator", value: "delivery-orchestrator"},
	{path: "grant.principal", value: "foundation-maintainer"},
	{path: "grant.failureOwnership", value: "foundation"},
	{path: "grant.purpose", value: "close copied-invocation replay, fetch-source replacement, and ambient production-executor bypasses"},
	{path: "grant.attemptId", value: "w001-lifecycle-ci-hardening-v17"},
	{path: "grant.autonomousMutation", value: "false"},
	{path: "grant.productionAllowed", value: "false"},
	{path: "grant.implementationAllowed", value: "true"},
	{path: "grant.canonicalLifecycleMutationAllowed", value: "false"},
	{path: "grant.developmentLeaseAllowed", value: "false"},
	{path: "findings.oneShotFinding", value: "the captured executor field could be copied and invoked outside the wrapper mutex and nil transition"},
	{path: "findings.sourceFinding", value: "the admitted local fetch source remained a pathname that could be replaced before Git consumed it"},
	{path: "findings.processFinding", value: "same-package tests could call the ambient arbitrary-argv production gitOutput executor outside the test constructor inventory"},
	{path: "findings.nextAction", value: "prospective-shared-one-shot-descriptor-stream-and-closed-production-executor-admission"},
	{path: "canonicalPreimage.bead", value: "M3-W001"},
	{path: "canonicalPreimage.nativeStatus", value: "in_progress"},
	{path: "canonicalPreimage.lifecycleState", value: "in-progress"},
	{path: "canonicalPreimage.workVersionGeneration", value: "6e79ff81-a007-42a5-a178-7ce58dbb718b"},
	{path: "canonicalPreimage.workVersionIncarnation", value: "e1e8d2d3f80871096a568fb489f49575a42abd37b269df9faf777a09cd689b41"},
	{path: "canonicalPreimage.issueMutationSequence", value: "1"},
	{path: "canonicalPreimage.dependencyGraphRevision", value: "1"},
	{path: "canonicalPreimage.liveLeaseState", value: "absent"},
	{path: "verification.publicCommitGateRequired", value: "true"},
	{path: "verification.immutableCommitReviewRequired", value: "true"},
	{path: "verification.protectedMainRequired", value: "true"},
	{path: "verification.externalBeadsReadbackRequired", value: "true"},
	{path: "verification.canonicalLifecycleMutationDeferred", value: "true"},
	{path: "integrity.signatureFormat", value: "openssh"},
	{path: "integrity.signatureNamespace", value: w001LifecycleCIHardeningV17Namespace},
	{path: "integrity.detachedSignature", value: "W-001-lifecycle-ci-hardening-v17.yaml.sig"},
	{path: "integrity.publicKey", value: "../keys/genesis-signing-key.pub"},
}

var w001LifecycleCIHardeningV17Sequences = map[string][]string{
	"grant.allowedEffects": {
		"preserve-the-v9-runtime-qualification-and-v10-v16-immutable-history-runs-and-dispositions",
		"move-the-one-shot-consumption-gate-inside-the-captured-executor-state-shared-by-all-copies",
		"descriptor-bind-both-fetch-source-and-destination-at-command-admission",
		"replace-fetch-path-consumption-with-fixed-source-pack-and-destination-index-transfer",
		"pass-no-admitted-fetch-source-pathname-to-any-child-process",
		"derive-the-complete-direct-production-process-entrypoint-inventory-from-non-test-source",
		"deny-every-production-process-entrypoint-identifier-and-direct-executor-field-access-from-tests",
		"retain-and-extend-all-v10-v16-environment-argv-process-and-physical-path-regressions",
		"update-only-the-plan-manifest-public-evidence-and-offline-validator-for-this-hardening",
		"reproduce-the-pinned-v9-native-Beads-artifact-and-non-skipped-conformance-without-changing-patch-bytes",
		"create-signed-semantic-commits-and-one-signed-v17-release-manager-review-tag",
		"push-the-existing-review-branch-and-tag-and-run-one-fresh-pull-request-10-gate",
		"obtain-fresh-independent-QA-and-Security-review-before-merge",
	},
	"grant.authorizedPaths": {
		w001LifecycleCIHardeningV17Path, w001LifecycleCIHardeningV17Signature,
		".harness/manifest.yaml", canonicalActivePlan, "docs/evidence/W-001-validation.md",
		"internal/doctrine/grant.go", "internal/doctrine/grant_test.go",
	},
	"grant.requiredProperties": {
		"v9-authority-runtime-contract-native-patches-product-contracts-and-qualification-bytes-remain-unchanged",
		"copied-invocation-values-and-extracted-executor-fields-share-one-non-resettable-consumption-gate",
		"fetch-source-and-destination-identities-are-descriptor-bound-before-the-invocation-is-returned",
		"no-source-pathname-is-consumed-after-fetch-admission-and-replacement-cannot-redirect-the-transfer",
		"every-transfer-process-uses-the-fixed-descriptor-trampoline-literal-system-executables-and-zero-ambient-environment",
		"direct-production-process-entrypoints-are-derived-repository-wide-and-none-is-callable-from-tests",
		"direct-executor-field-access-and-unexpected-process-callers-fail-closed-under-recursive-admission",
		"process-and-path-enumeration-fails-closed-on-unreadable-unparseable-or-symlinked-surfaces",
		"v16-QA-accepted-and-Security-changes-requested-dispositions-remain-durable",
		"the-next-public-run-uses-the-exact-signed-v17-tree-and-tag",
		"current-W001-lifecycle-and-live-lease-state-remain-unchanged",
	},
	"grant.prohibitedEffects": {
		"modify-authority-runtime-native-Beads-patches-database-schema-API-contract-or-product-contract",
		"mutate-M3-W001-or-any-other-Bead",
		"issue-assert-renew-release-or-revoke-a-canonical-live-lease",
		"rerun-or-move-any-v9-v10-v11-v12-v13-v14-v15-or-v16-commit-tag-or-run",
		"merge-pull-request-10-before-fresh-QA-and-Security-acceptance",
		"modify-workflow-scanner-ruleset-repository-settings-trust-roots-or-approval-policy",
		"expose-authority-credentials-raw-payloads-private-data-or-provider-state",
		"production-deployment-or-destructive-migration", "autonomous-mutation", "trust-escalation",
	},
	"findings.codes": {
		"ci.test_git_invocation_one_shot_field_bypass",
		"ci.test_git_fetch_source_toctou",
		"ci.test_process_guard_refresh_executor_bypass",
	},
	"verification.order": {"qa", "security-reviewer", "delivery-orchestrator"},
}

type strictPlanningGrant struct {
	scalars          map[string][]string
	sequences        map[string][]string
	sections         map[string]int
	sequenceHeaders  map[string]int
	structuralErrors []string
}

// CheckPlanningGrant validates the signed, public, contract-publication grant
// without consulting an operational authority or executing a signer process.
func CheckPlanningGrant(repo string) ([]Finding, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	checkWave1PlanningGrant(root, &findings)
	sortFindings(findings)
	return findings, nil
}

func checkWave1PlanningGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1PlanningGrantPath)
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_missing", "signed Wave-1 contract-publication grant is required")
		return
	}
	defer checkWave1PlanningGrantGitDiff(root, findings)

	document := parseStrictPlanningGrant(data)
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_schema", "%s", message)
	}
	for _, expected := range wave1PlanningGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_value", "%s does not match the signed Wave-1 contract", expected.path)
		}
	}
	for path, expected := range wave1PlanningGrantSequences {
		if document.sequenceHeaders[path] != 1 {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_sequence", "%s must occur exactly once", path)
			continue
		}
		actual := document.sequences[path]
		if !equalStringSequence(actual, expected) {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_sequence", "%s must equal the exact ordered Wave-1 contract", path)
		}
	}
	for _, section := range []string{"grant", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1PlanningGrantSignature)
	if signatureErr != nil {
		addFinding(findings, wave1PlanningGrantSignature, "public.planning_grant_signature_missing", "detached planning-grant signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if keyErr != nil {
		addFinding(findings, wave1PlanningGrantKey, "public.planning_grant_key_missing", "pinned genesis public key is required")
		return
	}

	keyValid := true
	if fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		keyValid = false
		addFinding(findings, wave1PlanningGrantKey, "public.planning_grant_key_anchor", "planning grant must use the independently pinned genesis key")
	}
	fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey)
	if fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
		addFinding(findings, wave1PlanningGrantKey, "public.planning_grant_key_fingerprint", "planning-grant signer fingerprint does not match the genesis authority")
	}
	if signatureErr == nil && keyValid {
		if err := verifySSHSig(data, signature, publicKey, wave1PlanningGrantNamespace); err != nil {
			addFinding(findings, wave1PlanningGrantSignature, "public.planning_grant_signature", "%v", err)
		}
	}
	checkWave1RecoveryDisposition(root, findings)
}

func checkWave1RecoveryDisposition(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1DispositionPath)
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.recovery_disposition_missing", "signed Wave-1 recovery disposition is required")
		return
	}
	document := parseStrictGrant(data, wave1DispositionScalars, wave1DispositionSequences, []string{"disposition", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1DispositionPath, "public.recovery_disposition_schema", "%s", message)
	}
	for _, expected := range wave1DispositionScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_value", "%s does not match the signed recovery contract", expected.path)
		}
	}
	for path, expected := range wave1DispositionSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_sequence", "%s must equal the exact ordered recovery contract", path)
		}
	}
	for _, section := range []string{"disposition", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1DispositionPath, "public.recovery_disposition_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1DispositionSignature)
	if signatureErr != nil {
		addFinding(findings, wave1DispositionSignature, "public.recovery_disposition_signature_missing", "detached recovery-disposition signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.recovery_disposition_key", "recovery disposition must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1DispositionNamespace); err != nil {
			addFinding(findings, wave1DispositionSignature, "public.recovery_disposition_signature", "%v", err)
		}
	}

	snapshot, snapshotErr := readRepoFile(root, wave1DispositionSnapshot)
	if snapshotErr != nil {
		addFinding(findings, wave1DispositionSnapshot, "public.recovery_snapshot_missing", "reconstructible public recovery snapshot is required")
	} else {
		if fileSHA256(snapshot) != wave1DispositionSnapshotSHA {
			addFinding(findings, wave1DispositionSnapshot, "public.recovery_snapshot_digest", "recovery snapshot does not match the signed SHA-256 binding")
		}
		var decoded map[string]any
		if err := decodeStrictJSON(snapshot, &decoded); err != nil {
			addFinding(findings, wave1DispositionSnapshot, "public.recovery_snapshot_schema", "recovery snapshot must be one strict JSON document")
		}
	}
	for _, legacy := range []string{
		".harness/grants/WAVE-1-authority-recovery.yaml",
		".harness/grants/WAVE-1-authority-recovery.yaml.sig",
	} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(legacy))); err == nil {
			addFinding(findings, legacy, "public.recovery_legacy_artifact", "scanner-triggering legacy recovery artifact must remain outside Git; the signed disposition binds its public checksum")
		} else if !os.IsNotExist(err) {
			addFinding(findings, legacy, "public.recovery_legacy_artifact", "legacy recovery artifact state cannot be established")
		}
	}
	checkWave1CIRecoveryAddendum(root, findings)
}

func checkWave1CIRecoveryAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1AddendumPath)
	if err != nil {
		addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_missing", "signed Wave-1 CI recovery addendum is required after the preserved failed publication")
		return
	}
	document := parseStrictGrant(data, wave1AddendumScalars, wave1AddendumSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_schema", "%s", message)
	}
	for _, expected := range wave1AddendumScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_value", "%s does not match the signed CI recovery contract", expected.path)
		}
	}
	for path, expected := range wave1AddendumSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_sequence", "%s must equal the exact ordered CI recovery contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1AddendumSignature)
	if signatureErr != nil {
		addFinding(findings, wave1AddendumSignature, "public.ci_recovery_addendum_signature_missing", "detached CI recovery addendum signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.ci_recovery_addendum_key", "CI recovery addendum must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1AddendumNamespace); err != nil {
			addFinding(findings, wave1AddendumSignature, "public.ci_recovery_addendum_signature", "%v", err)
		}
	}
	checkWave1V3CIRecoveryAddendum(root, findings)
}

func checkWave1V3CIRecoveryAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1V3AddendumPath)
	if err != nil {
		addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_missing", "signed Wave-1 v3 CI recovery addendum is required after the preserved stale-merge failure")
		return
	}
	document := parseStrictGrant(data, wave1V3AddendumScalars, wave1V3AddendumSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_schema", "%s", message)
	}
	for _, expected := range wave1V3AddendumScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_value", "%s does not match the signed v3 CI recovery contract", expected.path)
		}
	}
	for path, expected := range wave1V3AddendumSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_sequence", "%s must equal the exact ordered v3 CI recovery contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1V3AddendumSignature)
	if signatureErr != nil {
		addFinding(findings, wave1V3AddendumSignature, "public.ci_recovery_v3_addendum_signature_missing", "detached v3 CI recovery addendum signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.ci_recovery_v3_addendum_key", "v3 CI recovery addendum must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1V3AddendumNamespace); err != nil {
			addFinding(findings, wave1V3AddendumSignature, "public.ci_recovery_v3_addendum_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1DirectMainGrantPath))); err == nil {
		checkWave1DirectMainTransitionGrant(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_state", "direct-main transition grant state cannot be established")
	}
}

func checkWave1DirectMainTransitionGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1DirectMainGrantPath)
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_missing", "signed direct-main transition grant is required")
		return
	}
	document := parseStrictGrant(data, wave1DirectMainGrantScalars, wave1DirectMainGrantSequences, []string{"grant", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_schema", "%s", message)
	}
	for _, expected := range wave1DirectMainGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_value", "%s does not match the signed transition contract", expected.path)
		}
	}
	for path, expected := range wave1DirectMainGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_sequence", "%s must equal the exact ordered transition contract", path)
		}
	}
	for _, section := range []string{"grant", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_schema", "%s mapping must occur exactly once", section)
		}
	}

	signature, signatureErr := readRepoFile(root, wave1DirectMainGrantSignature)
	if signatureErr != nil {
		addFinding(findings, wave1DirectMainGrantSignature, "public.direct_main_transition_signature_missing", "detached direct-main transition signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.direct_main_transition_key", "direct-main transition must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1DirectMainGrantNamespace); err != nil {
			addFinding(findings, wave1DirectMainGrantSignature, "public.direct_main_transition_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1PRFallbackPath))); err == nil {
		checkWave1PRFallbackAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_state", "PR-fallback addendum state cannot be established")
	}
}

func checkWave1PRFallbackAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1PRFallbackPath)
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_missing", "signed PR-fallback addendum is required")
		return
	}
	document := parseStrictGrant(data, wave1PRFallbackScalars, wave1PRFallbackSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_schema", "%s", message)
	}
	for _, expected := range wave1PRFallbackScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_value", "%s does not match the signed fallback contract", expected.path)
		}
	}
	for path, expected := range wave1PRFallbackSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_sequence", "%s must equal the exact ordered fallback contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_schema", "%s mapping must occur exactly once", section)
		}
	}
	signature, signatureErr := readRepoFile(root, wave1PRFallbackSignature)
	if signatureErr != nil {
		addFinding(findings, wave1PRFallbackSignature, "public.pr_fallback_signature_missing", "detached PR-fallback signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_key", "PR fallback must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1PRFallbackNamespace); err != nil {
			addFinding(findings, wave1PRFallbackSignature, "public.pr_fallback_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1MainCIFixPath))); err == nil {
		checkWave1MainCIFixAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_state", "main-CI correction addendum state cannot be established")
	}
}

func checkWave1MainCIFixAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1MainCIFixPath)
	if err != nil {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_missing", "signed main-CI correction addendum is required")
		return
	}
	document := parseStrictGrant(data, wave1MainCIFixScalars, wave1MainCIFixSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_schema", "%s", message)
	}
	for _, expected := range wave1MainCIFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_value", "%s does not match the signed main-CI correction contract", expected.path)
		}
	}
	for path, expected := range wave1MainCIFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_sequence", "%s must equal the exact ordered main-CI correction contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_schema", "%s mapping must occur exactly once", section)
		}
	}
	signature, signatureErr := readRepoFile(root, wave1MainCIFixSignature)
	if signatureErr != nil {
		addFinding(findings, wave1MainCIFixSignature, "public.pr_fallback_main_ci_signature_missing", "detached main-CI correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_main_ci_key", "main-CI correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1MainCIFixNamespace); err != nil {
			addFinding(findings, wave1MainCIFixSignature, "public.pr_fallback_main_ci_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1CIFixtureFixPath))); err == nil {
		checkWave1CIFixtureFixAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_state", "fixture-stabilization addendum state cannot be established")
	}
}

func checkWave1CIFixtureFixAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, wave1CIFixtureFixPath)
	if err != nil {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_missing", "signed fixture-stabilization addendum is required")
		return
	}
	document := parseStrictGrant(data, wave1CIFixtureFixScalars, wave1CIFixtureFixSequences, []string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_schema", "%s", message)
	}
	for _, expected := range wave1CIFixtureFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_value", "%s does not match the signed fixture-stabilization contract", expected.path)
		}
	}
	for path, expected := range wave1CIFixtureFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_sequence", "%s must equal the exact ordered fixture-stabilization contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_schema", "%s mapping must occur exactly once", section)
		}
	}
	signature, signatureErr := readRepoFile(root, wave1CIFixtureFixSignature)
	if signatureErr != nil {
		addFinding(findings, wave1CIFixtureFixSignature, "public.pr_fallback_fixture_signature_missing", "detached fixture-stabilization signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_fixture_key", "fixture stabilization must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, wave1CIFixtureFixNamespace); err != nil {
			addFinding(findings, wave1CIFixtureFixSignature, "public.pr_fallback_fixture_signature", "%v", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001BootstrapGrantPath))); err == nil {
		checkW001BootstrapGrant(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_state", "W-001 bootstrap grant state cannot be established")
	}
}

func checkW001BootstrapGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001BootstrapGrantPath)
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_missing", "signed W-001 bootstrap grant is required")
		return
	}
	document := parseStrictGrant(data, w001BootstrapGrantScalars, w001BootstrapGrantSequences,
		[]string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_schema", "%s", message)
	}
	for _, expected := range w001BootstrapGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_value", "%s does not match the signed W-001 bootstrap contract", expected.path)
		}
	}
	for path, expected := range w001BootstrapGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_sequence", "%s must equal the exact ordered W-001 bootstrap contract", path)
		}
	}
	for _, section := range []string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_schema", "%s mapping must occur exactly once", section)
		}
	}

	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_expiry", "bootstrap grant must use one RFC3339 interval no longer than 72 hours")
	}
	incarnationInput, _ := json.Marshal(map[string]string{
		"created_at": scalarValue(document, "expected.createdAt"),
		"id":         scalarValue(document, "grant.bead"),
	})
	if fileSHA256(incarnationInput) != scalarValue(document, "postimage.issueIncarnation") {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_incarnation", "issue incarnation must be the signed digest of the canonical issue identity")
	}

	signature, signatureErr := readRepoFile(root, w001BootstrapGrantSignature)
	if signatureErr != nil {
		addFinding(findings, w001BootstrapGrantSignature, "public.w001_bootstrap_signature_missing", "detached W-001 bootstrap signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_bootstrap_key", "W-001 bootstrap grant must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001BootstrapGrantNamespace); err != nil {
			addFinding(findings, w001BootstrapGrantSignature, "public.w001_bootstrap_signature", "%v", err)
		}
	}

	securityCorrectionActive := false
	if _, stateErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); stateErr == nil {
		securityCorrectionActive = true
	} else if !os.IsNotExist(stateErr) {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_state", "postclaim Security-correction state cannot be established")
	}
	hookCorrectionActive := false
	if _, stateErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimHookFixPath))); stateErr == nil {
		hookCorrectionActive = true
	} else if !os.IsNotExist(stateErr) {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_state", "postclaim hook-isolation state cannot be established")
	}
	for _, binding := range []struct {
		pathField string
		hashField string
		code      string
	}{
		{"toolchain.patchPath", "toolchain.patchSHA256", "public.w001_bootstrap_patch_digest"},
		{"toolchain.helperCommandPath", "toolchain.helperCommandSHA256", "public.w001_bootstrap_helper_digest"},
		{"toolchain.helperLibraryPath", "toolchain.helperLibrarySHA256", "public.w001_bootstrap_helper_digest"},
	} {
		path := scalarValue(document, binding.pathField)
		expectedDigest := scalarValue(document, binding.hashField)
		if hookCorrectionActive && binding.pathField == "toolchain.helperLibraryPath" {
			expectedDigest = w001PostclaimHookHelperSHA
		} else if securityCorrectionActive && binding.pathField == "toolchain.helperLibraryPath" {
			expectedDigest = w001PostclaimSecurityHelperSHA
		}
		content, err := readRepoFile(root, path)
		if err != nil || !sha256Pattern.MatchString(expectedDigest) || fileSHA256(content) != expectedDigest {
			addFinding(findings, path, binding.code, "bootstrap helper material must match its exact signed SHA-256")
		}
	}

	postPayload, err := base64.RawStdEncoding.DecodeString(scalarValue(document, "postimage.metadataBase64"))
	if err != nil || !json.Valid(postPayload) || fileSHA256(postPayload) != scalarValue(document, "postimage.metadataSHA256") {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_postimage", "postimage metadata must be valid unpadded base64 JSON with the signed digest")
	} else {
		var metadata map[string]any
		if json.Unmarshal(postPayload, &metadata) != nil || metadata["lifecycleState"] != "in-progress" {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_postimage", "postimage metadata must set the exact lifecycle")
		}
		workVersion, ok := metadata["workVersion"].(map[string]any)
		if !ok || workVersion["authorityGeneration"] != scalarValue(document, "postimage.generationRef") ||
			workVersion["issueIncarnation"] != scalarValue(document, "postimage.issueIncarnation") ||
			workVersion["issueMutationSequence"] != float64(1) || workVersion["dependencyGraphRevision"] != float64(1) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_work_version", "postimage metadata must carry the exact bootstrap WorkVersion")
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimGrantPath))); err == nil {
		checkW001PostclaimGrant(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_state", "W-001 postclaim reconciliation state cannot be established")
	}
}

func checkW001PostclaimGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimGrantPath)
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_missing", "signed W-001 postclaim reconciliation grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimGrantScalars, w001PostclaimGrantSequences,
		[]string{"grant", "claim", "publication", "postimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_schema", "%s", message)
	}
	for _, expected := range w001PostclaimGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_value", "%s does not match the signed postclaim contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_sequence", "%s must equal the exact ordered postclaim contract", path)
		}
	}
	for _, section := range []string{"grant", "claim", "publication", "postimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_expiry", "postclaim grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimGrantSignature)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimGrantSignature, "public.w001_postclaim_signature_missing", "detached postclaim reconciliation signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_key", "postclaim reconciliation must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimGrantNamespace); err != nil {
			addFinding(findings, w001PostclaimGrantSignature, "public.w001_postclaim_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path    string
		digest  string
		code    string
		message string
	}{
		{w001BootstrapGrantPath, scalarValue(document, "grant.priorGrantSHA256"), "public.w001_postclaim_prior_grant", "prior bootstrap grant"},
		{w001BootstrapGrantSignature, scalarValue(document, "grant.priorGrantSignatureSHA256"), "public.w001_postclaim_prior_grant", "prior bootstrap signature"},
		{".harness/manifest.yaml", scalarValue(document, "postimage.manifestSHA256"), "public.w001_postclaim_postimage", "manifest postimage"},
		{canonicalActivePlan, scalarValue(document, "postimage.activePlanSHA256"), "public.w001_postclaim_postimage", "active-plan postimage"},
		{"docs/evidence/W-001-bootstrap-transition.md", scalarValue(document, "postimage.evidenceSHA256"), "public.w001_postclaim_postimage", "claim-evidence postimage"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if binding.path == ".harness/manifest.yaml" || binding.path == canonicalActivePlan {
			if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001DeliveryBase+":"+binding.path)
			}
		}
		// Additive successor grants never rewrite historical truth. Once v4 is
		// present, validate the v1 evidence bytes from the exact signed v3 Git
		// object while v4 independently validates the current evidence bytes.
		if binding.path == "docs/evidence/W-001-bootstrap-transition.md" {
			if _, successorErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimHookFixPath))); successorErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001PostclaimHookFixBase+":"+binding.path)
			}
		}
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "%s must match its exact signed SHA-256", binding.message)
		}
	}

	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-bootstrap-transition.md")
	receiptDigest, receiptErr := extractW001ClaimReceiptDigest(evidence)
	if evidenceErr != nil || receiptErr != nil || receiptDigest != scalarValue(document, "claim.receiptSHA256") {
		addFinding(findings, "docs/evidence/W-001-bootstrap-transition.md", "public.w001_postclaim_receipt", "canonical claim receipt must be bound by one exact public SHA-256 reference")
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimCIFixPath))); err == nil {
		checkW001PostclaimCIFixAddendum(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_state", "postclaim CI-stabilization state cannot be established")
	}
}

func checkW001PostclaimCIFixAddendum(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimCIFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_missing", "signed postclaim CI-stabilization addendum is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimCIFixScalars, w001PostclaimCIFixSequences,
		[]string{"addendum", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_schema", "%s", message)
	}
	for _, expected := range w001PostclaimCIFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_value", "%s does not match the signed CI-stabilization contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimCIFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_sequence", "%s must equal the exact ordered CI-stabilization contract", path)
		}
	}
	for _, section := range []string{"addendum", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "addendum.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "addendum.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_expiry", "CI-stabilization addendum must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimCIFixSignature)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimCIFixSignature, "public.w001_postclaim_ci_signature_missing", "detached CI-stabilization signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_ci_key", "CI stabilization must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimCIFixNamespace); err != nil {
			addFinding(findings, w001PostclaimCIFixSignature, "public.w001_postclaim_ci_signature", "%v", err)
		}
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001PostclaimGrantPath, scalarValue(document, "addendum.priorGrantSHA256")},
		{w001PostclaimGrantSignature, scalarValue(document, "addendum.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		// Preserve v5's evidence proof against the exact signed v5 tree when
		// the chronology successor corrects the current public record.
		if binding.path == "docs/evidence/W-001-validation.md" {
			if _, successorErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimChronoFixPath))); successorErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001PostclaimChronoFixBase+":"+binding.path)
			}
		}
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_postclaim_ci_prior_grant", "prior postclaim grant material must match its exact signed SHA-256")
		}
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); err == nil {
		checkW001PostclaimSecurityFix(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_state", "postclaim Security-correction state cannot be established")
	}
}

func checkW001PostclaimSecurityFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimSecurityFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_missing", "signed postclaim Security-correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimSecurityFixScalars, w001PostclaimSecurityFixSequences,
		[]string{"grant", "finding", "canonicalEffect", "materials", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_schema", "%s", message)
	}
	for _, expected := range w001PostclaimSecurityFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_value", "%s does not match the signed Security-correction contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimSecurityFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_sequence", "%s must equal the exact ordered Security-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "canonicalEffect", "materials", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_expiry", "Security-correction grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimSecurityFixSig)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimSecurityFixSig, "public.w001_postclaim_security_signature_missing", "detached Security-correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_security_key", "Security correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimSecurityFixNS); err != nil {
			addFinding(findings, w001PostclaimSecurityFixSig, "public.w001_postclaim_security_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
		code   string
	}{
		{w001PostclaimCIFixPath, scalarValue(document, "grant.priorAddendumSHA256"), "public.w001_postclaim_security_prior_addendum"},
		{w001PostclaimCIFixSignature, scalarValue(document, "grant.priorAddendumSignatureSHA256"), "public.w001_postclaim_security_prior_addendum"},
		{scalarValue(document, "materials.helperLibraryPath"), scalarValue(document, "materials.helperLibrarySHA256"), "public.w001_postclaim_security_material"},
		{scalarValue(document, "materials.helperTestPath"), scalarValue(document, "materials.helperTestSHA256"), "public.w001_postclaim_security_material"},
		{scalarValue(document, "materials.basePatchPath"), scalarValue(document, "materials.basePatchSHA256"), "public.w001_postclaim_security_material"},
		{scalarValue(document, "materials.securityPatchPath"), scalarValue(document, "materials.securityPatchSHA256"), "public.w001_postclaim_security_material"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		// Preserve the v3 material proof against its exact signed commit. The v4
		// validator separately binds the successor helper and test bytes.
		if binding.path == "internal/authority/bootstrap/bootstrap.go" || binding.path == "internal/authority/bootstrap/bootstrap_test.go" {
			if _, successorErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimHookFixPath))); successorErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001PostclaimHookFixBase+":"+binding.path)
			}
		}
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "Security-correction material must match its exact signed SHA-256")
		}
	}
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
		evidence, evidenceErr = planningGrantGitOutput(root, "show", w001DeliveryBase+":docs/evidence/W-001-validation.md")
	}
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("bootstrap-effective-database-selector-splice")) ||
		!bytes.Contains(evidence, []byte("**Current disposition:** changes-requested")) ||
		!bytes.Contains(evidence, []byte("earlier Security acceptance")) || !bytes.Contains(evidence, []byte("is superseded")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_postclaim_security_evidence", "Security correction evidence must preserve the exact additive supersession and finding fingerprint")
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimHookFixPath))); err == nil {
		checkW001PostclaimHookFix(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_state", "postclaim hook-isolation state cannot be established")
	}
}

func checkW001PostclaimHookFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimHookFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_missing", "signed postclaim hook-isolation grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimHookFixScalars, w001PostclaimHookFixSequences,
		[]string{"grant", "finding", "canonicalEffect", "materials", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_schema", "%s", message)
	}
	for _, expected := range w001PostclaimHookFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_value", "%s does not match the signed hook-isolation contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimHookFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_sequence", "%s must equal the exact ordered hook-isolation contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "canonicalEffect", "materials", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_expiry", "hook-isolation grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimHookFixSig)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimHookFixSig, "public.w001_postclaim_hook_signature_missing", "detached hook-isolation signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_hook_key", "hook-isolation correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimHookFixNS); err != nil {
			addFinding(findings, w001PostclaimHookFixSig, "public.w001_postclaim_hook_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
		code   string
	}{
		{w001PostclaimSecurityFixPath, scalarValue(document, "grant.priorGrantSHA256"), "public.w001_postclaim_hook_prior_grant"},
		{w001PostclaimSecurityFixSig, scalarValue(document, "grant.priorGrantSignatureSHA256"), "public.w001_postclaim_hook_prior_grant"},
		{scalarValue(document, "materials.validationEvidencePath"), scalarValue(document, "materials.validationEvidenceSHA256"), "public.w001_postclaim_hook_material"},
		{scalarValue(document, "materials.transitionEvidencePath"), scalarValue(document, "materials.transitionEvidenceSHA256"), "public.w001_postclaim_hook_material"},
		{scalarValue(document, "materials.helperLibraryPath"), scalarValue(document, "materials.helperLibrarySHA256"), "public.w001_postclaim_hook_material"},
		{scalarValue(document, "materials.helperTestPath"), scalarValue(document, "materials.helperTestSHA256"), "public.w001_postclaim_hook_material"},
		{scalarValue(document, "materials.basePatchPath"), scalarValue(document, "materials.basePatchSHA256"), "public.w001_postclaim_hook_material"},
		{scalarValue(document, "materials.securityPatchPath"), scalarValue(document, "materials.securityPatchSHA256"), "public.w001_postclaim_hook_material"},
		{scalarValue(document, "materials.hookIsolationPatchPath"), scalarValue(document, "materials.hookIsolationPatchSHA256"), "public.w001_postclaim_hook_material"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		// A publication-binding successor may correct only the current evidence
		// identity. Keep v4's material proof bound to the exact signed v4 tree.
		if binding.path == "docs/evidence/W-001-validation.md" {
			if _, successorErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimPRFixPath))); successorErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001PostclaimPRFixBase+":"+binding.path)
			}
		}
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "hook-isolation material must match its exact signed SHA-256")
		}
	}
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
		evidence, evidenceErr = planningGrantGitOutput(root, "show", w001DeliveryBase+":docs/evidence/W-001-validation.md")
	}
	transition, transitionErr := readRepoFile(root, "docs/evidence/W-001-bootstrap-transition.md")
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("bootstrap-workspace-hook-postcommit-effect")) ||
		!bytes.Contains(evidence, []byte("## v3 Security disposition")) ||
		!bytes.Contains(evidence, []byte("**Current disposition:** changes-requested")) ||
		transitionErr != nil || !bytes.Contains(transition, []byte("Canonical claim verified and reconciled; no live lease or implementation capability exists")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_postclaim_hook_evidence", "hook-isolation evidence must preserve the v3 disposition, exact finding fingerprint, and truthful canonical status")
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimPRFixPath))); err == nil {
		checkW001PostclaimPRFix(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_state", "postclaim publication-binding state cannot be established")
	}
}

func checkW001PostclaimPRFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimPRFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_missing", "signed postclaim publication-binding grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimPRFixScalars, w001PostclaimPRFixSequences,
		[]string{"grant", "finding", "publication", "materials", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_schema", "%s", message)
	}
	for _, expected := range w001PostclaimPRFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_value", "%s does not match the signed publication-binding contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimPRFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_sequence", "%s must equal the exact ordered publication-binding contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "publication", "materials", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_expiry", "publication-binding grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimPRFixSig)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimPRFixSig, "public.w001_postclaim_pr_binding_signature_missing", "detached publication-binding signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_pr_binding_key", "publication binding must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimPRFixNS); err != nil {
			addFinding(findings, w001PostclaimPRFixSig, "public.w001_postclaim_pr_binding_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
		code   string
	}{
		{w001PostclaimHookFixPath, scalarValue(document, "grant.priorGrantSHA256"), "public.w001_postclaim_pr_binding_prior_grant"},
		{w001PostclaimHookFixSig, scalarValue(document, "grant.priorGrantSignatureSHA256"), "public.w001_postclaim_pr_binding_prior_grant"},
		{scalarValue(document, "materials.validationEvidencePath"), scalarValue(document, "materials.validationEvidenceSHA256"), "public.w001_postclaim_pr_binding_material"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if binding.path == "docs/evidence/W-001-validation.md" {
			if _, successorErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimChronoFixPath))); successorErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001PostclaimChronoFixBase+":"+binding.path)
			}
		}
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "publication-binding material must match its exact signed SHA-256")
		}
	}
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
		evidence, evidenceErr = planningGrantGitOutput(root, "show", w001DeliveryBase+":docs/evidence/W-001-validation.md")
	}
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("stale-publication-vehicle-binding")) ||
		!bytes.Contains(evidence, []byte("PR #7 remains closed and unmerged historical evidence")) ||
		!bytes.Contains(evidence, []byte("PR #8 is the sole\nactive publication vehicle")) ||
		!bytes.Contains(evidence, []byte("PR #8 must remain unmerged")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_postclaim_pr_binding_evidence", "publication evidence must preserve PR 7 history and bind PR 8 as the sole active vehicle")
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimChronoFixPath))); err == nil {
		checkW001PostclaimChronoFix(root, findings)
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_state", "postclaim chronology-correction state cannot be established")
	}
}

func checkW001PostclaimChronoFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001PostclaimChronoFixPath)
	if err != nil {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_missing", "signed postclaim chronology-correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001PostclaimChronoFixScalars, w001PostclaimChronoFixSequences,
		[]string{"grant", "finding", "chronology", "publication", "materials", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_schema", "%s", message)
	}
	for _, expected := range w001PostclaimChronoFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_value", "%s does not match the signed chronology-correction contract", expected.path)
		}
	}
	for path, expected := range w001PostclaimChronoFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_sequence", "%s must equal the exact ordered chronology-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "chronology", "publication", "materials", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_expiry", "chronology-correction grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001PostclaimChronoFixSig)
	if signatureErr != nil {
		addFinding(findings, w001PostclaimChronoFixSig, "public.w001_postclaim_chronology_signature_missing", "detached chronology-correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_chronology_key", "chronology correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001PostclaimChronoFixNS); err != nil {
			addFinding(findings, w001PostclaimChronoFixSig, "public.w001_postclaim_chronology_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
		code   string
	}{
		{w001PostclaimPRFixPath, scalarValue(document, "grant.priorGrantSHA256"), "public.w001_postclaim_chronology_prior_grant"},
		{w001PostclaimPRFixSig, scalarValue(document, "grant.priorGrantSignatureSHA256"), "public.w001_postclaim_chronology_prior_grant"},
		{scalarValue(document, "materials.validationEvidencePath"), scalarValue(document, "materials.validationEvidenceSHA256"), "public.w001_postclaim_chronology_material"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if binding.path == "docs/evidence/W-001-validation.md" {
			if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
				content, readErr = planningGrantGitOutput(root, "show", w001DeliveryBase+":"+binding.path)
			}
		}
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, binding.code, "chronology-correction material must match its exact signed SHA-256")
		}
	}
	checkW001PostclaimChronology(root, document, issuedAt, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
		evidence, evidenceErr = planningGrantGitOutput(root, "show", w001DeliveryBase+":docs/evidence/W-001-validation.md")
	}
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("grant-effective-after-governed-effects")) ||
		!bytes.Contains(evidence, []byte("not retroactively relabelled as\nauthorized")) ||
		!bytes.Contains(evidence, []byte("complete reviewed tree through a new signed commit, tag, PR #8 run")) ||
		!bytes.Contains(evidence, []byte("signed v6 tree")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_postclaim_chronology_evidence", "chronology evidence must preserve the incident and compensating v6 publication route")
	}
	if _, deliveryErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); deliveryErr == nil {
		checkW001DeliveryGrant(root, findings)
	} else if !os.IsNotExist(deliveryErr) {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_state", "W-001 delivery-grant state cannot be established")
	}
}

func checkW001DeliveryGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001DeliveryGrantPath)
	if err != nil {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_missing", "signed W-001 delivery grant is required")
		return
	}
	document := parseStrictGrant(data, w001DeliveryGrantScalars, w001DeliveryGrantSequences,
		[]string{"grant", "canonicalPreimage", "publication", "reconciliation", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_schema", "%s", message)
	}
	for _, expected := range w001DeliveryGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_value", "%s does not match the signed delivery contract", expected.path)
		}
	}
	for path, expected := range w001DeliveryGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_sequence", "%s must equal the exact ordered delivery contract", path)
		}
	}
	for _, section := range []string{"grant", "canonicalPreimage", "publication", "reconciliation", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_expiry", "delivery grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001DeliveryGrantSignature)
	if signatureErr != nil {
		addFinding(findings, w001DeliveryGrantSignature, "public.w001_delivery_signature_missing", "detached W-001 delivery signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_delivery_key", "delivery grant must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001DeliveryGrantNamespace); err != nil {
			addFinding(findings, w001DeliveryGrantSignature, "public.w001_delivery_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001PostclaimChronoFixPath, scalarValue(document, "grant.priorGrantSHA256")},
		{w001PostclaimChronoFixSig, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_delivery_prior_grant", "prior postclaim material must match its exact signed SHA-256")
		}
	}
	if !checkW001DeliveryPriorTag(root, findings) {
		return
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001DeliveryBase+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001DeliveryBase+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001DeliveryBase || strings.TrimSpace(string(baseTree)) != w001DeliveryBaseTree {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_base", "delivery must descend from the exact accepted postclaim squash and tree")
	}
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	priorDispositionPresent := bytes.Contains(evidence, []byte("**Current disposition:** postclaim reconciliation accepted, merged, and completed")) ||
		bytes.Contains(evidence, []byte("**Historical disposition:** postclaim reconciliation accepted, merged, and completed"))
	if evidenceErr != nil || !priorDispositionPresent ||
		!bytes.Contains(evidence, []byte("01a0408e-ca08-71f0-b1ac-0dec0039706a")) ||
		!bytes.Contains(evidence, []byte(scalarValue(document, "reconciliation.commentSHA256"))) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_delivery_evidence", "delivery evidence must bind the exact completed merge and Beads reconciliation receipt")
	}
	if planErr != nil || !bytes.Contains(plan, []byte("Delivery authority: signed grant `W-001-delivery-v2`")) ||
		!bytes.Contains(plan, []byte("w001-delivery-87d9680d-ca5a-4f3d-9afc-741884232e73")) {
		addFinding(findings, canonicalActivePlan, "public.w001_delivery_plan", "active plan must select the exact signed delivery grant and attempt")
	}
	lifecycleActive := false
	if _, lifecycleErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleGrantPath))); lifecycleErr == nil {
		lifecycleActive = true
	} else if !os.IsNotExist(lifecycleErr) {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_state", "lifecycle-completion grant state cannot be established")
	}
	if !lifecycleActive && (manifestErr != nil || !bytes.Contains(manifest, []byte("active_delivery_grant: W-001-delivery-v2")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-delivery-87d9680d-ca5a-4f3d-9afc-741884232e73")) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent"))) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_delivery_manifest", "manifest must project the exact delivery grant, attempt, and absent initial lease")
	}
	if _, correctionErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryCIFixPath))); correctionErr == nil {
		checkW001DeliveryCIFix(root, findings)
	} else if !os.IsNotExist(correctionErr) {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_state", "delivery CI-correction state cannot be established")
	}
	if _, scannerErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryScannerFixPath))); scannerErr == nil {
		checkW001DeliveryScannerFix(root, findings)
	} else if !os.IsNotExist(scannerErr) {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_state", "delivery scanner-correction state cannot be established")
	}
	if lifecycleActive {
		checkW001LifecycleCompletionGrant(root, findings)
	}
}

func checkW001DeliveryCIFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001DeliveryCIFixPath)
	if err != nil {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_missing", "signed W-001 delivery CI correction is required")
		return
	}
	document := parseStrictGrant(data, w001DeliveryCIFixScalars, w001DeliveryCIFixSequences,
		[]string{"grant", "finding", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_schema", "%s", message)
	}
	for _, expected := range w001DeliveryCIFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_value", "%s does not match the signed CI-correction contract", expected.path)
		}
	}
	for path, expected := range w001DeliveryCIFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_sequence", "%s must equal the exact ordered CI-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_expiry", "CI-correction grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001DeliveryCIFixSignature)
	if signatureErr != nil {
		addFinding(findings, w001DeliveryCIFixSignature, "public.w001_delivery_ci_signature_missing", "detached delivery CI-correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_delivery_ci_key", "CI correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001DeliveryCIFixNamespace); err != nil {
			addFinding(findings, w001DeliveryCIFixSignature, "public.w001_delivery_ci_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001DeliveryGrantPath, scalarValue(document, "grant.priorGrantSHA256")},
		{w001DeliveryGrantSignature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_delivery_ci_prior_grant", "prior delivery material must match its exact signed SHA-256")
		}
	}
	checkW001DeliveryV2Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("delivery-review-tag/release-identity-mismatch")) ||
		!bytes.Contains(evidence, []byte("98454619462")) || !bytes.Contains(evidence, []byte("98454903898")) ||
		!bytes.Contains(evidence, []byte("The retry budget is exhausted; the run will not be retried again.")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_delivery_ci_evidence", "delivery CI evidence must preserve the normalized failure and exhausted retry")
	}
}

func checkW001DeliveryScannerFix(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001DeliveryScannerFixPath)
	if err != nil {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_missing", "signed W-001 delivery scanner correction is required")
		return
	}
	document := parseStrictGrant(data, w001DeliveryScannerFixScalars, w001DeliveryScannerFixSequences,
		[]string{"grant", "finding", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_schema", "%s", message)
	}
	for _, expected := range w001DeliveryScannerFixScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_value", "%s does not match the signed scanner-correction contract", expected.path)
		}
	}
	for path, expected := range w001DeliveryScannerFixSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_sequence", "%s must equal the exact ordered scanner-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_expiry", "scanner-correction grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001DeliveryScannerFixSignature)
	if signatureErr != nil {
		addFinding(findings, w001DeliveryScannerFixSignature, "public.w001_delivery_scanner_signature_missing", "detached scanner-correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_delivery_scanner_key", "scanner correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001DeliveryScannerFixNamespace); err != nil {
			addFinding(findings, w001DeliveryScannerFixSignature, "public.w001_delivery_scanner_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001DeliveryCIFixPath, scalarValue(document, "grant.priorGrantSHA256")},
		{w001DeliveryCIFixSignature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_delivery_scanner_prior_grant", "prior delivery correction must match its exact signed SHA-256")
		}
	}
	checkW001DeliveryV3Tag(root, findings)
	checkW001DeliveryScannerIgnore(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("delivery-history-scanner/preserved-v1-synthetic-generic-key")) ||
		!bytes.Contains(evidence, []byte("33066374068")) || !bytes.Contains(evidence, []byte("98497338894")) ||
		!bytes.Contains(evidence, []byte("Adding a new committed\nsynthetic credential canary still produced one `github-pat` finding")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_delivery_scanner_evidence", "scanner evidence must preserve the exact history failure and new-canary proof")
	}
}

func checkW001LifecycleCompletionGrant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001LifecycleGrantPath)
	if err != nil {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_missing", "signed W-001 lifecycle-completion grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleGrantScalars, w001LifecycleGrantSequences,
		[]string{"grant", "finding", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_schema", "%s", message)
	}
	for _, expected := range w001LifecycleGrantScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_value", "%s does not match the signed lifecycle-completion contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleGrantSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_sequence", "%s must equal the exact ordered lifecycle-completion contract", path)
		}
	}
	for _, section := range []string{"grant", "finding", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_expiry", "lifecycle-completion grant must use one RFC3339 interval no longer than 72 hours")
	}

	signature, signatureErr := readRepoFile(root, w001LifecycleGrantSignature)
	if signatureErr != nil {
		addFinding(findings, w001LifecycleGrantSignature, "public.w001_lifecycle_signature_missing", "detached lifecycle-completion signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_key", "lifecycle completion must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001LifecycleGrantNamespace); err != nil {
			addFinding(findings, w001LifecycleGrantSignature, "public.w001_lifecycle_signature", "%v", err)
		}
	}

	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001DeliveryScannerFixPath, scalarValue(document, "grant.priorGrantSHA256")},
		{w001DeliveryScannerFixSignature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_prior_grant", "prior delivery correction must match its exact signed SHA-256")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleBase+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleBase+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleBase || strings.TrimSpace(string(baseTree)) != w001LifecycleBaseTree {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_base", "lifecycle completion must descend from the exact accepted core squash and tree")
	}
	checkW001LifecyclePriorTag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if evidenceErr != nil || !bytes.Contains(evidence, []byte("completion-audit/governed-lifecycle-routes-missing")) ||
		!bytes.Contains(evidence, []byte("33069887434/98509103754")) ||
		!bytes.Contains(evidence, []byte("d7ddb1c0d4ecb00b93fcbec4d56b740da581a725e91e6381601d2d295203c38d")) {
		addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_evidence", "lifecycle evidence must preserve the accepted core merge and exact completion finding")
	}
	correctionActive := false
	if _, correctionErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionPath))); correctionErr == nil {
		correctionActive = true
	} else if !os.IsNotExist(correctionErr) {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_state", "lifecycle correction state cannot be established")
	}
	if planErr != nil || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) ||
		(!correctionActive && !bytes.Contains(plan, []byte("`W-001-lifecycle-completion-v5` correction"))) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_plan", "active plan must select the truthful lifecycle-completion correction")
	}
	if manifestErr != nil || (!correctionActive && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-completion-v5")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-completion-v5")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_manifest", "manifest must project the lifecycle-completion attempt and absent live lease")
	}
	if correctionActive {
		checkW001LifecycleCorrectionGrant(root, findings)
	}
}

func checkW001LifecycleCorrectionGrant(root string, findings *[]Finding) {
	v7Active := false
	if _, v7Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV7Path))); v7Err == nil {
		v7Active = true
	} else if !os.IsNotExist(v7Err) {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_state", "v7 lifecycle correction state cannot be established")
	}
	data, err := readRepoFile(root, w001LifecycleCorrectionPath)
	if err != nil {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_missing", "signed W-001 lifecycle correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCorrectionScalars, w001LifecycleCorrectionSequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCorrectionScalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_value", "%s does not match the signed lifecycle-correction contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCorrectionSequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_sequence", "%s must equal the exact ordered lifecycle-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_expiry", "lifecycle correction grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCorrectionSignature)
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCorrectionSignature, "public.w001_lifecycle_correction_signature_missing", "detached lifecycle correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_correction_key", "lifecycle correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001LifecycleCorrectionNamespace); err != nil {
			addFinding(findings, w001LifecycleCorrectionSignature, "public.w001_lifecycle_correction_signature", "%v", err)
		}
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleGrantPath, scalarValue(document, "grant.priorGrantSHA256")},
		{w001LifecycleGrantSignature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_correction_prior_grant", "prior lifecycle material must match its exact signed SHA-256")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionBase+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionBase+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCorrectionBase || strings.TrimSpace(string(baseTree)) != w001LifecycleCorrectionBaseTree {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_base", "lifecycle correction must descend from the exact reviewed v5 head and tree")
	}
	checkW001LifecycleV5Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	for _, marker := range []string{
		"lifecycle.terminal_claim_binding_fail_open",
		"lifecycle.handoff_replay_fence_splice",
		"lifecycle.missing_receipt_replay_success",
		"lifecycle.nonterminal_convergence_deadlock",
		"lifecycle.qualification_not_reproducible",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_correction_evidence", "lifecycle correction evidence must preserve all exact v5 findings")
			break
		}
	}
	if planErr != nil || (!v7Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-correction-v6`"))) ||
		!bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_correction_plan", "active plan must select the truthful v6 lifecycle correction")
	}
	if manifestErr != nil || (!v7Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-correction-v6")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-correction-v6")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_correction_manifest", "manifest must project the v6 lifecycle correction and absent live lease")
	}
	if v7Active {
		checkW001LifecycleCorrectionV7Grant(root, findings)
	}
}

func checkW001LifecycleCorrectionV7Grant(root string, findings *[]Finding) {
	v8Active := false
	if _, v8Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV8Path))); v8Err == nil {
		v8Active = true
	} else if !os.IsNotExist(v8Err) {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_state", "v8 lifecycle correction state cannot be established")
	}
	data, err := readRepoFile(root, w001LifecycleCorrectionV7Path)
	if err != nil {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_missing", "signed v7 lifecycle correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCorrectionV7Scalars, w001LifecycleCorrectionV7Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCorrectionV7Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_value", "%s does not match the signed v7 lifecycle-correction contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCorrectionV7Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_sequence", "%s must equal the exact ordered v7 lifecycle-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_expiry", "v7 lifecycle correction grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCorrectionV7Signature)
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCorrectionV7Signature, "public.w001_lifecycle_correction_v7_signature_missing", "detached v7 lifecycle correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_correction_v7_key", "v7 lifecycle correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001LifecycleCorrectionV7Namespace); err != nil {
			addFinding(findings, w001LifecycleCorrectionV7Signature, "public.w001_lifecycle_correction_v7_signature", "%v", err)
		}
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCorrectionPath, scalarValue(document, "grant.priorGrantSHA256")},
		{w001LifecycleCorrectionSignature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_correction_v7_prior_grant", "prior v6 lifecycle material must match its exact signed SHA-256")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionV7Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionV7Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCorrectionV7Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCorrectionV7BaseTree {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_base", "v7 lifecycle correction must descend from the exact reviewed v6 head and tree")
	}
	checkW001LifecycleV6Tag(root, findings)
	checkW001LifecycleCorrectionV7Evidence(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v8Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-correction-v7`"))) ||
		!bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_correction_v7_plan", "active plan must select the truthful v7 lifecycle correction")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v8Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-correction-v7")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-correction-v7")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_correction_v7_manifest", "manifest must project the v7 lifecycle correction and absent live lease")
	}
	if v8Active {
		checkW001LifecycleCorrectionV8Grant(root, findings)
	}
}

func checkW001LifecycleCorrectionV7Evidence(root string, findings *[]Finding) {
	evidencePath := "docs/evidence/W-001-validation.md"
	evidence, evidenceErr := readRepoFile(root, evidencePath)
	for _, marker := range []string{
		"lifecycle.claim_lineage_not_joined",
		"lifecycle.failure_fingerprint_retry_not_monotonic",
		"lifecycle.qualification_not_independently_reproducible",
		w001LifecycleCorrectionV7PatchSHA,
		"70dfa9e28546dc0c6dbe8046f5960577514b0bc6793fda98b3383931a50a72d8",
		"sha256:b1a0cc29a7e13e0595e21087eeb930dc494976b18ba68279bf52c665f3170aa0",
		"bb8dd437802943670b4e882a3cdc30d5ea5a3b2035171fb765d7d82db7f624de",
		"canonical-claim attempt on every current",
		"third occurrence is rejected",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, evidencePath, "public.w001_lifecycle_correction_v7_evidence", "v7 lifecycle evidence must preserve the exact findings and qualification bindings")
			break
		}
	}
	patch, patchErr := readRepoFile(root, w001LifecycleCorrectionV7PatchPath)
	if _, v8Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV8Path))); v8Err == nil {
		patch, patchErr = planningGrantGitOutput(root, "show", w001LifecycleCorrectionV8Base+":"+w001LifecycleCorrectionV7PatchPath)
	}
	if patchErr != nil || fileSHA256(patch) != w001LifecycleCorrectionV7PatchSHA {
		addFinding(findings, w001LifecycleCorrectionV7PatchPath, "public.w001_lifecycle_correction_v7_patch", "v7 lifecycle patch must match its exact reviewed SHA-256")
	}
}

func checkW001LifecycleCorrectionV8Grant(root string, findings *[]Finding) {
	v9Active := false
	if _, v9Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV9Path))); v9Err == nil {
		v9Active = true
	} else if !os.IsNotExist(v9Err) {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_state", "v9 lifecycle correction state cannot be established")
	}
	data, err := readRepoFile(root, w001LifecycleCorrectionV8Path)
	if err != nil {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_missing", "signed v8 lifecycle correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCorrectionV8Scalars, w001LifecycleCorrectionV8Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCorrectionV8Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_value", "%s does not match the signed v8 lifecycle-correction contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCorrectionV8Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_sequence", "%s must equal the exact ordered v8 lifecycle-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_expiry", "v8 lifecycle correction grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCorrectionV8Signature)
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCorrectionV8Signature, "public.w001_lifecycle_correction_v8_signature_missing", "detached v8 lifecycle correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_correction_v8_key", "v8 lifecycle correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001LifecycleCorrectionV8Namespace); err != nil {
			addFinding(findings, w001LifecycleCorrectionV8Signature, "public.w001_lifecycle_correction_v8_signature", "%v", err)
		}
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCorrectionV7Path, scalarValue(document, "grant.priorGrantSHA256")},
		{w001LifecycleCorrectionV7Signature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_correction_v8_prior_grant", "prior v7 lifecycle material must match its exact signed SHA-256")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionV8Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionV8Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCorrectionV8Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCorrectionV8BaseTree {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_base", "v8 lifecycle correction must descend from the exact reviewed v7 head and tree")
	}
	checkW001LifecycleV7Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"lifecycle.claim_key_alias_not_canonical",
		"lifecycle.legacy_active_scalar_contradiction",
		"lifecycle.dependency_detailed_state_ignored",
		w001LifecycleCorrectionV8Base,
		w001LifecycleCorrectionV8BaseTree,
		w001LifecycleCorrectionV8PatchSHA,
		"5fb4120f30c9d54d4dd847755a8070d305c1a7a14b783e7ce33157b432b02665",
		"a478f5090ca1b616e5aa8e5b74f4277814a8f0b1a88d990f9b7876761a3a7cc7",
		"case-fold claim alias rejection",
		"Dependency projections decode the full canonical metadata",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_correction_v8_evidence", "v8 lifecycle evidence must preserve the exact v7 findings")
			break
		}
	}
	patch, patchErr := readRepoFile(root, w001LifecycleCorrectionV7PatchPath)
	if v9Active {
		patch, patchErr = planningGrantGitOutput(root, "show", w001LifecycleCorrectionV9Base+":"+w001LifecycleCorrectionV7PatchPath)
	}
	if patchErr != nil || fileSHA256(patch) != w001LifecycleCorrectionV8PatchSHA {
		addFinding(findings, w001LifecycleCorrectionV7PatchPath, "public.w001_lifecycle_correction_v8_patch", "v8 lifecycle patch must match its exact reviewed SHA-256")
	}
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v9Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-correction-v8`"))) ||
		!bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_correction_v8_plan", "active plan must select the truthful v8 lifecycle correction")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v9Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-correction-v8")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-correction-v8")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_correction_v8_manifest", "manifest must project the v8 lifecycle correction and absent live lease")
	}
	if v9Active {
		checkW001LifecycleCorrectionV9Grant(root, findings)
	}
}

func checkW001LifecycleCorrectionV9Grant(root string, findings *[]Finding) {
	v10Active := false
	if _, v10Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleStabilizationV10Path))); v10Err == nil {
		v10Active = true
	} else if !os.IsNotExist(v10Err) {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_state", "v10 lifecycle CI stabilization state cannot be established")
	}
	data, err := readRepoFile(root, w001LifecycleCorrectionV9Path)
	if err != nil {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_missing", "signed v9 lifecycle correction grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCorrectionV9Scalars, w001LifecycleCorrectionV9Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCorrectionV9Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_value", "%s does not match the signed v9 lifecycle-correction contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCorrectionV9Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_sequence", "%s must equal the exact ordered v9 lifecycle-correction contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_expiry", "v9 lifecycle correction grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCorrectionV9Signature)
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCorrectionV9Signature, "public.w001_lifecycle_correction_v9_signature_missing", "detached v9 lifecycle correction signature is required")
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_correction_v9_key", "v9 lifecycle correction must use the independently pinned genesis key")
	} else if signatureErr == nil {
		if err := verifySSHSig(data, signature, publicKey, w001LifecycleCorrectionV9Namespace); err != nil {
			addFinding(findings, w001LifecycleCorrectionV9Signature, "public.w001_lifecycle_correction_v9_signature", "%v", err)
		}
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCorrectionV8Path, scalarValue(document, "grant.priorGrantSHA256")},
		{w001LifecycleCorrectionV8Signature, scalarValue(document, "grant.priorGrantSignatureSHA256")},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || !sha256Pattern.MatchString(binding.digest) || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_correction_v9_prior_grant", "prior v8 lifecycle material must match its exact signed SHA-256")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionV9Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCorrectionV9Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCorrectionV9Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCorrectionV9BaseTree {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_base", "v9 lifecycle correction must descend from the exact reviewed v8 head and tree")
	}
	checkW001LifecycleV8Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"lifecycle.native_recursive_key_alias_not_canonical",
		"lifecycle.dependency_lineage_stripping",
		w001LifecycleCorrectionV9Base,
		w001LifecycleCorrectionV9BaseTree,
		w001LifecycleCorrectionV9PatchSHA,
		"91b3e8dd5c8c0c01b5953c4c38ca508a150b05cd719f4e80fec293365afddf7f",
		"recursive canonical-key rejection",
		"sparse legacy dependency compatibility",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_correction_v9_evidence", "v9 lifecycle evidence must preserve the exact v8 findings and correction materials")
			break
		}
	}
	patch, patchErr := readRepoFile(root, w001LifecycleCorrectionV7PatchPath)
	if patchErr != nil || fileSHA256(patch) != w001LifecycleCorrectionV9PatchSHA {
		addFinding(findings, w001LifecycleCorrectionV7PatchPath, "public.w001_lifecycle_correction_v9_patch", "v9 lifecycle patch must match its exact reviewed SHA-256")
	}
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v10Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-correction-v9`"))) ||
		!bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_correction_v9_plan", "active plan must select the truthful v9 lifecycle correction")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v10Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-correction-v9")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-correction-v9")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_correction_v9_manifest", "manifest must project the v9 lifecycle correction and absent live lease")
	}
	if v10Active {
		checkW001LifecycleStabilizationV10Grant(root, findings)
	}
}

func checkW001LifecycleStabilizationV10Grant(root string, findings *[]Finding) {
	v11Active := false
	if _, v11Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIFencingV11Path))); v11Err == nil {
		v11Active = true
	} else if !os.IsNotExist(v11Err) {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_state", "v11 lifecycle CI fencing state cannot be established")
	}
	if v11Active {
		defer checkW001LifecycleStabilizationV10Successor(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleStabilizationV10Path)
	if err != nil {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_missing", "signed v10 lifecycle CI stabilization grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleStabilizationV10Scalars, w001LifecycleStabilizationV10Sequences,
		[]string{"grant", "failure", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_schema", "%s", message)
	}
	for _, expected := range w001LifecycleStabilizationV10Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_value", "%s does not match the signed v10 lifecycle CI stabilization contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleStabilizationV10Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_sequence", "%s must equal the exact ordered v10 lifecycle CI stabilization contract", path)
		}
	}
	for _, section := range []string{"grant", "failure", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_expiry", "v10 lifecycle CI stabilization grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleStabilizationV10Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleStabilizationV10Signature, "public.w001_lifecycle_stabilization_v10_signature_missing", "detached v10 lifecycle CI stabilization signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_stabilization_v10_key", "v10 lifecycle CI stabilization must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleStabilizationV10Namespace); err != nil {
		addFinding(findings, w001LifecycleStabilizationV10Signature, "public.w001_lifecycle_stabilization_v10_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCorrectionV9Path, "9e3d37216650b81f552862beff1224ada762cfbc177d5b043c9785f65579b984"},
		{w001LifecycleCorrectionV9Signature, "4bb3a1cad372bc891e89a9df709f771c4f4ac3bbdf35a639a653f27d5b147a3d"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_stabilization_v10_prior_grant", "prior v9 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleStabilizationV10Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleStabilizationV10Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleStabilizationV10Base || strings.TrimSpace(string(baseTree)) != w001LifecycleStabilizationV10BaseTree {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_base", "v10 lifecycle CI stabilization must descend from the exact immutable v9 head and tree")
	}
	checkW001LifecycleV9Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci/doctrine-tempdir-git-pack-cleanup", "33104553091", "98630789458", "98631170195",
		w001LifecycleStabilizationV10Base, w001LifecycleStabilizationV10BaseTree,
		"exhausted two identical retries", "no authority runtime bytes changed",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_stabilization_v10_evidence", "v10 evidence must preserve both v9 CI failures and the bounded stabilization")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	for _, marker := range []string{"maintenance.auto=false", "gc.auto=0", "gc.autoDetach=false", "maintenance.autoDetach=false", "TestPlanningGrantTestGitCommandDisablesBackgroundMaintenance"} {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_stabilization_v10_fixture", "every disposable Git command must carry the exact bounded maintenance configuration")
			break
		}
	}
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v11Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-stabilization-v10`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_stabilization_v10_plan", "active plan must select the truthful v10 CI stabilization")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v11Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-stabilization-v10")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-stabilization-v10")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_stabilization_v10_manifest", "manifest must project the v10 CI stabilization and absent live lease")
	}
}

func checkW001LifecycleStabilizationV10Successor(root string, findings *[]Finding) {
	checkW001LifecycleCIFencingV11Grant(root, findings)
}

func checkW001LifecycleCIFencingV11Grant(root string, findings *[]Finding) {
	v12Active := false
	if _, v12Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV12Path))); v12Err == nil {
		v12Active = true
	} else if !os.IsNotExist(v12Err) {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_state", "v12 lifecycle CI hardening state cannot be established")
	}
	if v12Active {
		defer checkW001LifecycleCIHardeningV12Grant(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleCIFencingV11Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_missing", "signed v11 lifecycle CI fencing grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIFencingV11Scalars, w001LifecycleCIFencingV11Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIFencingV11Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_value", "%s does not match the signed v11 lifecycle CI fencing contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIFencingV11Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_sequence", "%s must equal the exact ordered v11 lifecycle CI fencing contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_expiry", "v11 lifecycle CI fencing grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIFencingV11Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIFencingV11Signature, "public.w001_lifecycle_ci_fencing_v11_signature_missing", "detached v11 lifecycle CI fencing signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_fencing_v11_key", "v11 lifecycle CI fencing must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIFencingV11Namespace); err != nil {
		addFinding(findings, w001LifecycleCIFencingV11Signature, "public.w001_lifecycle_ci_fencing_v11_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleStabilizationV10Path, "b6f29734dabbeaff52f96c8d5e0a8910fadf81c841f4d9fd4a7cd799add586f9"},
		{w001LifecycleStabilizationV10Signature, "9bc96e8c5a0f35fee3998066bd6fa00b45b671ee1b66debad4f6e21c7341ab32"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_fencing_v11_prior_grant", "prior v10 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIFencingV11Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIFencingV11Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIFencingV11Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIFencingV11BaseTree {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_base", "v11 lifecycle CI fencing must descend from the exact immutable v10 head and tree")
	}
	checkW001LifecycleV10Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_sanitization_incomplete", "ci.test_git_configuration_persisted", w001LifecycleCIFencingV11Base,
		w001LifecycleCIFencingV11BaseTree, "33105792480", "98635155160", "command-local", "changes-requested",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_fencing_v11_evidence", "v11 evidence must preserve the exact v10 findings and bounded correction")
			break
		}
	}
	if !v12Active {
		tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
		rawNeedle := []byte("exec.Command(" + "\"git\"")
		if testsErr != nil || bytes.Count(tests, rawNeedle) != 1 || bytes.Contains(tests, []byte("disablePlanningGrantTestGitMaintenance")) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_fencing_v11_wrapper", "all disposable Git operations must use the single audited bounded wrapper without persistent fixture config")
		}
		for _, marker := range []string{"TestPlanningGrantTestGitCommandDisablesBackgroundMaintenance", "TestPlanningGrantDisposableGitCallsUseOnlyBoundedWrapper", "TestPlanningGrantGitFixtureDoesNotPersistMaintenanceConfiguration", "--local", "--global"} {
			if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
				addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_fencing_v11_fixture", "v11 test fencing regressions are required")
				break
			}
		}
	}
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v12Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-fencing-v11`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_fencing_v11_plan", "active plan must select the truthful v11 CI fencing correction")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v12Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-fencing-v11")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-fencing-v11")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_fencing_v11_manifest", "manifest must project the v11 CI fencing correction and absent live lease")
	}
}

func checkW001LifecycleCIHardeningV12Grant(root string, findings *[]Finding) {
	v13Active := false
	if _, v13Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV13Path))); v13Err == nil {
		v13Active = true
	} else if !os.IsNotExist(v13Err) {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_state", "v13 lifecycle CI hardening state cannot be established")
	}
	if v13Active {
		defer checkW001LifecycleCIHardeningV13Grant(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleCIHardeningV12Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_missing", "signed v12 lifecycle CI hardening grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIHardeningV12Scalars, w001LifecycleCIHardeningV12Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIHardeningV12Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_value", "%s does not match the signed v12 lifecycle CI hardening contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIHardeningV12Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_sequence", "%s must equal the exact ordered v12 lifecycle CI hardening contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_expiry", "v12 lifecycle CI hardening grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIHardeningV12Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIHardeningV12Signature, "public.w001_lifecycle_ci_hardening_v12_signature_missing", "detached v12 lifecycle CI hardening signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_hardening_v12_key", "v12 lifecycle CI hardening must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIHardeningV12Namespace); err != nil {
		addFinding(findings, w001LifecycleCIHardeningV12Signature, "public.w001_lifecycle_ci_hardening_v12_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCIFencingV11Path, "6df1dc4978e6b3657986ef43a41aaa3437567772c95bec4f151d8abcf0e9396b"},
		{w001LifecycleCIFencingV11Signature, "967efc83af964fc5abaf42f28cad1ad0231dc35524eecf5f9a05eae093d80b0e"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_hardening_v12_prior_grant", "prior v11 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV12Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV12Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIHardeningV12Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIHardeningV12BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_base", "v12 lifecycle CI hardening must descend from the exact immutable v11 head and tree")
	}
	checkW001LifecycleV11Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_fences_caller_overridable", "ci.test_git_environment_execution_injection", "ci.test_process_guard_fail_open",
		w001LifecycleCIHardeningV12Base, w001LifecycleCIHardeningV12BaseTree, "33108126981", "98643418071",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_hardening_v12_evidence", "v12 evidence must preserve the exact v11 findings and bounded hardening")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	testGitExecutorMarker := "exec.Command(\"/usr/bin/git\""
	if _, v16Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV16Path))); v16Err == nil {
		testGitExecutorMarker = "exec.Command(\"/usr/bin/perl\""
	}
	for _, marker := range []string{
		testGitExecutorMarker, "validatePlanningGrantTestGitArguments", "TestPlanningGrantTestGitArgumentsFailClosed",
		"TestPlanningGrantTestGitCommandRejectsAmbientExecutionInjection", "TestPlanningGrantTestProcessInvocationsFailClosedRepositoryWide",
	} {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v12_fixture", "v12 Git and process hardening regressions are required")
			break
		}
	}
	checkPlanningGrantTestProcessInvocations(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v13Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-hardening-v12`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_hardening_v12_plan", "active plan must select the truthful v12 CI hardening")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v13Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-hardening-v12")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-hardening-v12")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_hardening_v12_manifest", "manifest must project the v12 CI hardening and absent live lease")
	}
}

func checkW001LifecycleCIHardeningV13Grant(root string, findings *[]Finding) {
	v14Active := false
	if _, v14Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV14Path))); v14Err == nil {
		v14Active = true
	} else if !os.IsNotExist(v14Err) {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_state", "v14 lifecycle CI hardening state cannot be established")
	}
	if v14Active {
		defer checkW001LifecycleCIHardeningV14Grant(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleCIHardeningV13Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_missing", "signed v13 lifecycle CI hardening grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIHardeningV13Scalars, w001LifecycleCIHardeningV13Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIHardeningV13Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_value", "%s does not match the signed v13 lifecycle CI hardening contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIHardeningV13Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_sequence", "%s must equal the exact ordered v13 lifecycle CI hardening contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_expiry", "v13 lifecycle CI hardening grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIHardeningV13Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIHardeningV13Signature, "public.w001_lifecycle_ci_hardening_v13_signature_missing", "detached v13 lifecycle CI hardening signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_hardening_v13_key", "v13 lifecycle CI hardening must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIHardeningV13Namespace); err != nil {
		addFinding(findings, w001LifecycleCIHardeningV13Signature, "public.w001_lifecycle_ci_hardening_v13_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCIHardeningV12Path, "9356f21a72ce652b6238be15f6393cf402c34408fe467526e43f2a53c7ca5ab1"},
		{w001LifecycleCIHardeningV12Signature, "733e67f5ec4807523ab56febc60c21aed3f4a2cfe58253febc58cef3b5d113d1"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_hardening_v13_prior_grant", "prior v12 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV13Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV13Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIHardeningV13Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIHardeningV13BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_base", "v13 lifecycle CI hardening must descend from the exact immutable v12 head and tree")
	}
	checkW001LifecycleV12Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_argv_schema_fail_open", "ci.test_process_guard_incomplete",
		w001LifecycleCIHardeningV13Base, w001LifecycleCIHardeningV13BaseTree, "33110339883", "98651204635",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_hardening_v13_evidence", "v13 evidence must preserve the exact v12 findings and bounded hardening")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	for _, marker := range []string{
		"validatePlanningGrantTestGitSubcommand", "--upload-p=/tmp/hostile-upload-pack", "--separate-git-dir=/tmp/outside", "direct_exec_cmd", "indirect_syscall", "nested_exec_cmd",
	} {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v13_fixture", "v13 closed Git argv and recursive process regressions are required")
			break
		}
	}
	checkPlanningGrantTestProcessInvocations(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v14Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-hardening-v13`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_hardening_v13_plan", "active plan must select the truthful v13 CI hardening")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v14Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-hardening-v13")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-hardening-v13")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_hardening_v13_manifest", "manifest must project the v13 CI hardening and absent live lease")
	}
}

func checkW001LifecycleCIHardeningV14Grant(root string, findings *[]Finding) {
	v15Active := false
	if _, v15Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV15Path))); v15Err == nil {
		v15Active = true
	} else if !os.IsNotExist(v15Err) {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_state", "v15 lifecycle CI hardening state cannot be established")
	}
	if v15Active {
		defer checkW001LifecycleCIHardeningV15Grant(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleCIHardeningV14Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_missing", "signed v14 lifecycle CI hardening grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIHardeningV14Scalars, w001LifecycleCIHardeningV14Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIHardeningV14Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_value", "%s does not match the signed v14 lifecycle CI hardening contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIHardeningV14Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_sequence", "%s must equal the exact ordered v14 lifecycle CI hardening contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_expiry", "v14 lifecycle CI hardening grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIHardeningV14Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIHardeningV14Signature, "public.w001_lifecycle_ci_hardening_v14_signature_missing", "detached v14 lifecycle CI hardening signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_hardening_v14_key", "v14 lifecycle CI hardening must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIHardeningV14Namespace); err != nil {
		addFinding(findings, w001LifecycleCIHardeningV14Signature, "public.w001_lifecycle_ci_hardening_v14_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCIHardeningV13Path, "221545ade5766928436cf75c30250c32780305f5e7d12f8a60bb4d35659d109d"},
		{w001LifecycleCIHardeningV13Signature, "0984d7e54f0f93e3514c0ecb3cdd0cd5716d05a7b4571068648b47abcadf056f"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_hardening_v14_prior_grant", "prior v13 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV14Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV14Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIHardeningV14Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIHardeningV14BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_base", "v14 lifecycle CI hardening must descend from the exact immutable v13 head and tree")
	}
	checkW001LifecycleV13Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_clone_physical_escape", "ci.test_process_guard_transitive_bypass",
		w001LifecycleCIHardeningV14Base, w001LifecycleCIHardeningV14BaseTree, "33112938711", "98660186954",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_hardening_v14_evidence", "v14 evidence must preserve the exact v13 findings and bounded correction")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	physicalRegression := "TestPlanningGrantTestGitCommandRejectsSymlinkedCloneTargets"
	if v15Active {
		physicalRegression = "TestPlanningGrantTestGitCommandBindsCanonicalRootDescriptor"
	}
	for _, marker := range []string{
		physicalRegression, "production process entrypoint",
		"TestPlanningGrantTestProcessInvocationsRejectProductionGitExecutor",
	} {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v14_fixture", "v14 physical clone and production-executor regressions are required")
			break
		}
	}
	checkPlanningGrantTestProcessInvocations(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v15Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-hardening-v14`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_hardening_v14_plan", "active plan must select the truthful v14 CI hardening")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v15Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-hardening-v14")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-hardening-v14")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_hardening_v14_manifest", "manifest must project the v14 CI hardening and absent live lease")
	}
}

func checkW001LifecycleCIHardeningV15Grant(root string, findings *[]Finding) {
	v16Active := false
	if _, v16Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV16Path))); v16Err == nil {
		v16Active = true
	} else if !os.IsNotExist(v16Err) {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_state", "v16 lifecycle CI hardening state cannot be established")
	}
	if v16Active {
		defer checkW001LifecycleCIHardeningV16Grant(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleCIHardeningV15Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_missing", "signed v15 lifecycle CI hardening grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIHardeningV15Scalars, w001LifecycleCIHardeningV15Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIHardeningV15Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_value", "%s does not match the signed v15 lifecycle CI hardening contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIHardeningV15Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_sequence", "%s must equal the exact ordered v15 lifecycle CI hardening contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_expiry", "v15 lifecycle CI hardening grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIHardeningV15Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIHardeningV15Signature, "public.w001_lifecycle_ci_hardening_v15_signature_missing", "detached v15 lifecycle CI hardening signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_hardening_v15_key", "v15 lifecycle CI hardening must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIHardeningV15Namespace); err != nil {
		addFinding(findings, w001LifecycleCIHardeningV15Signature, "public.w001_lifecycle_ci_hardening_v15_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCIHardeningV14Path, "a9dd38a11cf0f076b8af79618739a303e69a63b41caca1c2fb23f8ada87e3eac"},
		{w001LifecycleCIHardeningV14Signature, "98d439533d5754ea7e52c66de0ff3ffe3348daf1aa6754038f16c68f2493cded"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_hardening_v15_prior_grant", "prior v14 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV15Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV15Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIHardeningV15Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIHardeningV15BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_base", "v15 lifecycle CI hardening must descend from the exact immutable v14 head and tree")
	}
	checkW001LifecycleV14Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_root_ancestor_alias_admitted", "ci.test_git_clone_reservation_toctou", "ci.test_process_guard_dot_import_bypass",
		w001LifecycleCIHardeningV15Base, w001LifecycleCIHardeningV15BaseTree, "33123061855", "98694494697",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_hardening_v15_evidence", "v15 evidence must preserve the exact v14 findings and bounded correction")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	testMarkers := []string{
		"TestPlanningGrantTestGitCommandBindsCanonicalRootDescriptor", "runPlanningGrantTestGitDescriptorHelper",
		"planningGrantTestBeforeGitProcess", "syscall.Fchdir", "dot_import_os", "dot_import_os_exec", "dot_import_syscall",
		"blank_import_os", "blank_import_os_exec", "blank_import_syscall",
	}
	if v16Active {
		testMarkers = []string{
			"TestPlanningGrantTestGitCommandBindsCanonicalRootDescriptor", "planningGrantTestGitDescriptorTrampoline",
			"planningGrantTestBeforeGitProcess", "removed_descriptor_helper", "dot_import_os", "dot_import_os_exec", "dot_import_syscall",
			"blank_import_os", "blank_import_os_exec", "blank_import_syscall",
		}
	}
	for _, marker := range testMarkers {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v15_fixture", "v15 descriptor-bound Git and closed process-import regressions are required")
			break
		}
	}
	if testsErr == nil && (bytes.Contains(tests, []byte(`case "clone":`)) || bytes.Contains(tests, []byte(`runPlanningGrantTestGit(t, root, "clone"`))) {
		addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v15_clone", "v15 doctrine test Git wrapper must not admit or execute clone")
	}
	checkPlanningGrantTestProcessInvocations(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v16Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-hardening-v15`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_hardening_v15_plan", "active plan must select the truthful v15 CI hardening")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v16Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-hardening-v15")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-hardening-v15")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_hardening_v15_manifest", "manifest must project the v15 CI hardening and absent live lease")
	}
}

func checkW001LifecycleCIHardeningV16Grant(root string, findings *[]Finding) {
	v17Active := false
	if _, v17Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV17Path))); v17Err == nil {
		v17Active = true
	} else if !os.IsNotExist(v17Err) {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_state", "v17 lifecycle CI hardening state cannot be established")
	}
	if v17Active {
		defer checkW001LifecycleCIHardeningV17Grant(root, findings)
	}
	data, err := readRepoFile(root, w001LifecycleCIHardeningV16Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_missing", "signed v16 lifecycle CI hardening grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIHardeningV16Scalars, w001LifecycleCIHardeningV16Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIHardeningV16Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_value", "%s does not match the signed v16 lifecycle CI hardening contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIHardeningV16Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_sequence", "%s must equal the exact ordered v16 lifecycle CI hardening contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_expiry", "v16 lifecycle CI hardening grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIHardeningV16Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIHardeningV16Signature, "public.w001_lifecycle_ci_hardening_v16_signature_missing", "detached v16 lifecycle CI hardening signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_hardening_v16_key", "v16 lifecycle CI hardening must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIHardeningV16Namespace); err != nil {
		addFinding(findings, w001LifecycleCIHardeningV16Signature, "public.w001_lifecycle_ci_hardening_v16_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCIHardeningV15Path, "b751be17403e0b41f75d853e0d8a4e6baa61101436a94bf61c47b42366ac409b"},
		{w001LifecycleCIHardeningV15Signature, "de2e0c92114e3aa4a4639206caa8a97d1ec39e8ab819989c7335c0df107e458e"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_hardening_v16_prior_grant", "prior v15 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV16Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV16Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIHardeningV16Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIHardeningV16BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_base", "v16 lifecycle CI hardening must descend from the exact immutable v15 head and tree")
	}
	checkW001LifecycleV15Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_descriptor_helper_transitive_bypass", "ci.test_git_helper_executable_path_toctou",
		w001LifecycleCIHardeningV16Base, w001LifecycleCIHardeningV16BaseTree, "33165311496", "98829194619",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_hardening_v16_evidence", "v16 evidence must preserve the exact v15 findings and bounded correction")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	testMarkers := []string{
		"planningGrantTestGitDescriptorTrampoline = `open(my $root, '<&=3') or exit 126; chdir($root) or exit 126; close($root) or exit 126; exec {'/usr/bin/git'} '/usr/bin/git', @ARGV; exit 126;`",
		"perlArguments := []string{\"-f\", \"-e\", planningGrantTestGitDescriptorTrampoline, \"--\"}",
		"exec.Command(\"/usr/bin/perl\"", "command.ExtraFiles = []*os.File{rootHandle}", "command.Env = planningGrantTestGitEnvironment()",
		"validatePlanningGrantTestGitFetchSource", "GIT_ALLOW_PROTOCOL=file", "removed_descriptor_helper", "dynamic_executable",
	}
	if v17Active {
		testMarkers = []string{
			"planningGrantTestGitDescriptorTrampoline = `open(my $root, '<&=3') or exit 126; chdir($root) or exit 126; close($root) or exit 126; exec {'/usr/bin/git'} '/usr/bin/git', @ARGV; exit 126;`",
			"exec.Command(\"/usr/bin/perl\"", "command.ExtraFiles = []*os.File{handle}", "command.Env = planningGrantTestGitEnvironment()",
			"admitPlanningGrantTestGitFetchSource", "GIT_ALLOW_PROTOCOL=file", "removed_descriptor_helper", "dynamic_executable",
		}
	}
	for _, marker := range testMarkers {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v16_fixture", "v16 fixed descriptor trampoline and closed process regressions are required")
			break
		}
	}
	for _, forbidden := range []string{
		"func TestMain(", "func runPlanningGrantTestGitDescriptorHelper(", "os.Executable()", "os.Environ()", "command.Env = planningGrantGitEnvironment()",
	} {
		if testsErr == nil && bytes.Contains(tests, []byte(forbidden)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v16_helper", "v16 must remove the self-executable helper and environment-selected TestMain mode")
			break
		}
	}
	checkPlanningGrantTestProcessInvocations(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || (!v17Active && !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-hardening-v16`"))) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_hardening_v16_plan", "active plan must select the truthful v16 CI hardening")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || (!v17Active && (!bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-hardening-v16")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-hardening-v16")))) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_hardening_v16_manifest", "manifest must project the v16 CI hardening and absent live lease")
	}
}

func checkW001LifecycleCIHardeningV17Grant(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001LifecycleCIHardeningV17Path)
	if err != nil {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_missing", "signed v17 lifecycle CI hardening grant is required")
		return
	}
	document := parseStrictGrant(data, w001LifecycleCIHardeningV17Scalars, w001LifecycleCIHardeningV17Sequences,
		[]string{"grant", "findings", "canonicalPreimage", "verification", "integrity"})
	for _, message := range document.structuralErrors {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_schema", "%s", message)
	}
	for _, expected := range w001LifecycleCIHardeningV17Scalars {
		values := document.scalars[expected.path]
		switch {
		case len(values) != 1:
			addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_field", "%s must occur exactly once", expected.path)
		case values[0] != expected.value:
			addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_value", "%s does not match the signed v17 lifecycle CI hardening contract", expected.path)
		}
	}
	for path, expected := range w001LifecycleCIHardeningV17Sequences {
		if document.sequenceHeaders[path] != 1 || !equalStringSequence(document.sequences[path], expected) {
			addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_sequence", "%s must equal the exact ordered v17 lifecycle CI hardening contract", path)
		}
	}
	for _, section := range []string{"grant", "findings", "canonicalPreimage", "verification", "integrity"} {
		if document.sections[section] != 1 {
			addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_schema", "%s mapping must occur exactly once", section)
		}
	}
	issuedAt, issueErr := time.Parse(time.RFC3339, scalarValue(document, "grant.issuedAt"))
	expiresAt, expiryErr := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if issueErr != nil || expiryErr != nil || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > 72*time.Hour {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_expiry", "v17 lifecycle CI hardening grant must use one RFC3339 interval no longer than 72 hours")
	}
	signature, signatureErr := readRepoFile(root, w001LifecycleCIHardeningV17Signature)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if signatureErr != nil {
		addFinding(findings, w001LifecycleCIHardeningV17Signature, "public.w001_lifecycle_ci_hardening_v17_signature_missing", "detached v17 lifecycle CI hardening signature is required")
	} else if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_ci_hardening_v17_key", "v17 lifecycle CI hardening must use the independently pinned genesis key")
	} else if err := verifySSHSig(data, signature, publicKey, w001LifecycleCIHardeningV17Namespace); err != nil {
		addFinding(findings, w001LifecycleCIHardeningV17Signature, "public.w001_lifecycle_ci_hardening_v17_signature", "%v", err)
	}
	for _, binding := range []struct {
		path   string
		digest string
	}{
		{w001LifecycleCIHardeningV16Path, "95fa2caa2befd270ed15f9c317a37ceec442b70c0826a869323d34c3d612d835"},
		{w001LifecycleCIHardeningV16Signature, "26711612175f7969ec168159e535e7b6b7273641690ed4c486e609cddaa844e5"},
	} {
		content, readErr := readRepoFile(root, binding.path)
		if readErr != nil || fileSHA256(content) != binding.digest {
			addFinding(findings, binding.path, "public.w001_lifecycle_ci_hardening_v17_prior_grant", "prior v16 lifecycle material must remain byte-exact")
		}
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV17Base+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleCIHardeningV17Base+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleCIHardeningV17Base || strings.TrimSpace(string(baseTree)) != w001LifecycleCIHardeningV17BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_base", "v17 lifecycle CI hardening must descend from the exact immutable v16 head and tree")
	}
	checkW001LifecycleV16Tag(root, findings)
	evidence, evidenceErr := readRepoFile(root, "docs/evidence/W-001-validation.md")
	for _, marker := range []string{
		"ci.test_git_invocation_one_shot_field_bypass", "ci.test_git_fetch_source_toctou", "ci.test_process_guard_refresh_executor_bypass",
		w001LifecycleCIHardeningV17Base, w001LifecycleCIHardeningV17BaseTree, "33206197037", "98967743138",
	} {
		if evidenceErr != nil || !bytes.Contains(evidence, []byte(marker)) {
			addFinding(findings, "docs/evidence/W-001-validation.md", "public.w001_lifecycle_ci_hardening_v17_evidence", "v17 evidence must preserve the exact v16 review findings and bounded correction")
			break
		}
	}
	tests, testsErr := readRepoFile(root, "internal/doctrine/grant_test.go")
	for _, marker := range []string{
		"TestPlanningGrantTestGitFetchBindsSourceDescriptor", "pack-objects", "index-pack", "admitPlanningGrantTestGitFetchSource",
		"one-shot invocation under contention", "direct_executor_field", "TestPlanningGrantTestProcessInventoryFailsClosed",
	} {
		if testsErr != nil || !bytes.Contains(tests, []byte(marker)) {
			addFinding(findings, "internal/doctrine/grant_test.go", "public.w001_lifecycle_ci_hardening_v17_fixture", "v17 shared one-shot, descriptor-stream fetch, and closed production process regressions are required")
			break
		}
	}
	checkPlanningGrantTestProcessInvocations(root, findings)
	plan, planErr := readRepoFile(root, canonicalActivePlan)
	if planErr != nil || !bytes.Contains(plan, []byte("`W-001-lifecycle-ci-hardening-v17`")) || !bytes.Contains(plan, []byte("W-001 therefore remains `in-progress`")) {
		addFinding(findings, canonicalActivePlan, "public.w001_lifecycle_ci_hardening_v17_plan", "active plan must select the truthful v17 CI hardening")
	}
	manifest, manifestErr := readRepoFile(root, ".harness/manifest.yaml")
	if manifestErr != nil || !bytes.Contains(manifest, []byte("active_delivery_grant: W-001-lifecycle-ci-hardening-v17")) ||
		!bytes.Contains(manifest, []byte("active_attempt: w001-lifecycle-ci-hardening-v17")) ||
		!bytes.Contains(manifest, []byte("live_lease_state: absent")) {
		addFinding(findings, ".harness/manifest.yaml", "public.w001_lifecycle_ci_hardening_v17_manifest", "manifest must project the v17 CI hardening and absent live lease")
	}
}

func checkPlanningGrantTestProcessInvocations(root string, findings *[]Finding) {
	var paths []string
	var productionPaths []string
	doctrineRoot := filepath.Join(root, "internal", "doctrine")
	walkErr := filepath.Walk(doctrineRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked doctrine test surface is not admitted: %s", path)
		}
		if info.Mode().IsRegular() && strings.HasSuffix(info.Name(), ".go") {
			if strings.HasSuffix(info.Name(), "_test.go") {
				paths = append(paths, path)
			} else {
				productionPaths = append(productionPaths, path)
			}
		}
		return nil
	})
	productionEntrypoints, productionErr := planningGrantProductionProcessEntrypoints(doctrineRoot, productionPaths)
	if walkErr != nil || len(paths) == 0 || productionErr != nil {
		addFinding(findings, "internal/doctrine", "public.w001_lifecycle_ci_process_guard", "doctrine test process-invocation surface cannot be enumerated")
		return
	}
	sort.Strings(paths)
	expected := map[string][]string{
		"grant_test.go:TestVerifyPlanningGrantTagRequiresExactSignedTreeAttestation": {"ssh-keygen", "ssh-keygen"},
		"grant_test.go:planningGrantTestGitCommand":                                  {"/usr/bin/perl"},
	}
	expectedCmdTypes := map[string]int{}
	actual := make(map[string][]string)
	actualCmdTypes := make(map[string]int)
	executorFieldSelectors := make(map[string]int)
	guardFailed := false
	for _, path := range paths {
		fileSet := token.NewFileSet()
		file, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			guardFailed = true
			continue
		}
		execAliases := make(map[string]bool)
		osAliases := make(map[string]bool)
		syscallAliases := make(map[string]bool)
		for _, spec := range file.Imports {
			importPath, unquoteErr := strconv.Unquote(spec.Path.Value)
			if unquoteErr != nil {
				guardFailed = true
				continue
			}
			alias := ""
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			switch importPath {
			case "os/exec":
				if alias == "" {
					alias = "exec"
				}
				if alias == "." || alias == "_" {
					guardFailed = true
				} else {
					execAliases[alias] = true
				}
			case "os":
				if alias == "" {
					alias = "os"
				}
				if alias == "." || alias == "_" {
					guardFailed = true
				} else {
					osAliases[alias] = true
				}
			case "syscall":
				if alias == "" {
					alias = "syscall"
				}
				if alias == "." || alias == "_" {
					guardFailed = true
				} else {
					syscallAliases[alias] = true
				}
			}
		}
		fileExecSelectorCount := 0
		functionExecSelectorCount := 0
		fileExecCmdTypeCount := 0
		functionExecCmdTypeCount := 0
		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.Ident:
				if productionEntrypoints[typed.Name] || typed.Name == "runPlanningGrantTestGitDescriptorHelper" {
					guardFailed = true
				}
			case *ast.SelectorExpr:
				identifier, ok := typed.X.(*ast.Ident)
				if ok && execAliases[identifier.Name] && (typed.Sel.Name == "Command" || typed.Sel.Name == "CommandContext") {
					fileExecSelectorCount++
				}
				if ok && execAliases[identifier.Name] && typed.Sel.Name == "Cmd" {
					fileExecCmdTypeCount++
				}
				if ok && osAliases[identifier.Name] && typed.Sel.Name == "StartProcess" {
					guardFailed = true
				}
				if ok && syscallAliases[identifier.Name] && (typed.Sel.Name == "Exec" || typed.Sel.Name == "ForkExec" || typed.Sel.Name == "StartProcess") {
					guardFailed = true
				}
			}
			return true
		})
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			relative, relErr := filepath.Rel(doctrineRoot, path)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				guardFailed = true
				continue
			}
			key := filepath.ToSlash(relative) + ":" + function.Name.Name
			execSelectorCount := 0
			execCallCount := 0
			execCmdTypeCount := 0
			ast.Inspect(function, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.SelectorExpr:
					if typed.Sel.Name == "combinedOutput" {
						executorFieldSelectors[key]++
						if key != "grant_test.go:CombinedOutput" {
							guardFailed = true
						}
					}
					identifier, isIdentifier := typed.X.(*ast.Ident)
					if isIdentifier && execAliases[identifier.Name] && (typed.Sel.Name == "Command" || typed.Sel.Name == "CommandContext") {
						execSelectorCount++
					}
					if isIdentifier && execAliases[identifier.Name] && typed.Sel.Name == "Cmd" {
						execCmdTypeCount++
					}
					if isIdentifier && osAliases[identifier.Name] && typed.Sel.Name == "StartProcess" {
						guardFailed = true
					}
					if isIdentifier && syscallAliases[identifier.Name] && (typed.Sel.Name == "Exec" || typed.Sel.Name == "ForkExec" || typed.Sel.Name == "StartProcess") {
						guardFailed = true
					}
				case *ast.CallExpr:
					selector, isSelector := typed.Fun.(*ast.SelectorExpr)
					if !isSelector {
						return true
					}
					identifier, isIdentifier := selector.X.(*ast.Ident)
					if !isIdentifier {
						return true
					}
					if execAliases[identifier.Name] {
						execCallCount++
						binary := "<dynamic>"
						if len(typed.Args) > 0 {
							if literal, ok := typed.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
								if value, unquoteErr := strconv.Unquote(literal.Value); unquoteErr == nil {
									binary = value
								}
							}
						}
						if binary == "<dynamic>" {
							guardFailed = true
						}
						actual[key] = append(actual[key], selector.Sel.Name+":"+binary)
					}
					if osAliases[identifier.Name] && selector.Sel.Name == "StartProcess" {
						guardFailed = true
					}
					if syscallAliases[identifier.Name] && (selector.Sel.Name == "Exec" || selector.Sel.Name == "ForkExec" || selector.Sel.Name == "StartProcess") {
						guardFailed = true
					}
				}
				return true
			})
			if execSelectorCount != execCallCount {
				guardFailed = true
			}
			functionExecSelectorCount += execSelectorCount
			functionExecCmdTypeCount += execCmdTypeCount
			if execCmdTypeCount > 0 {
				actualCmdTypes[key] = execCmdTypeCount
			}
		}
		if fileExecSelectorCount != functionExecSelectorCount {
			guardFailed = true
		}
		if fileExecCmdTypeCount != functionExecCmdTypeCount {
			guardFailed = true
		}
	}
	for key, calls := range actual {
		want, ok := expected[key]
		if !ok || len(calls) != len(want) {
			guardFailed = true
			continue
		}
		for index, call := range calls {
			if call != "Command:"+want[index] {
				guardFailed = true
			}
		}
	}
	for key, want := range expected {
		if len(actual[key]) != len(want) {
			guardFailed = true
		}
	}
	for key, count := range actualCmdTypes {
		if expectedCmdTypes[key] != count {
			guardFailed = true
		}
	}
	for key, count := range expectedCmdTypes {
		if actualCmdTypes[key] != count {
			guardFailed = true
		}
	}
	if len(executorFieldSelectors) != 1 || executorFieldSelectors["grant_test.go:CombinedOutput"] != 1 {
		guardFailed = true
	}
	if guardFailed {
		addFinding(findings, "internal/doctrine", "public.w001_lifecycle_ci_process_guard", "test process invocations must equal the exact recursive AST allowlist: calls=%v cmdTypes=%v", actual, actualCmdTypes)
	}
}

func planningGrantProductionProcessEntrypoints(doctrineRoot string, paths []string) (map[string]bool, error) {
	if len(paths) == 0 {
		return nil, errors.New("doctrine production process surface is empty")
	}
	sort.Strings(paths)
	actual := make(map[string]bool)
	for _, path := range paths {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return nil, err
		}
		execAliases := make(map[string]bool)
		osAliases := make(map[string]bool)
		syscallAliases := make(map[string]bool)
		for _, spec := range file.Imports {
			importPath, unquoteErr := strconv.Unquote(spec.Path.Value)
			if unquoteErr != nil {
				return nil, unquoteErr
			}
			alias := ""
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			switch importPath {
			case "os/exec":
				if alias == "" {
					alias = "exec"
				}
				if alias == "." || alias == "_" {
					return nil, errors.New("dot or blank production os/exec import is not admitted")
				}
				execAliases[alias] = true
			case "os":
				if alias == "" {
					alias = "os"
				}
				if alias == "." || alias == "_" {
					return nil, errors.New("dot or blank production os import is not admitted")
				}
				osAliases[alias] = true
			case "syscall":
				if alias == "" {
					alias = "syscall"
				}
				if alias == "." || alias == "_" {
					return nil, errors.New("dot or blank production syscall import is not admitted")
				}
				syscallAliases[alias] = true
			}
		}
		fileSelectors := 0
		functionSelectors := 0
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			if execAliases[identifier.Name] && (selector.Sel.Name == "Command" || selector.Sel.Name == "CommandContext" || selector.Sel.Name == "Cmd") ||
				osAliases[identifier.Name] && selector.Sel.Name == "StartProcess" ||
				syscallAliases[identifier.Name] && (selector.Sel.Name == "Exec" || selector.Sel.Name == "ForkExec" || selector.Sel.Name == "StartProcess") {
				fileSelectors++
			}
			return true
		})
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			count := 0
			ast.Inspect(function, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				identifier, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if execAliases[identifier.Name] && (selector.Sel.Name == "Command" || selector.Sel.Name == "CommandContext" || selector.Sel.Name == "Cmd") ||
					osAliases[identifier.Name] && selector.Sel.Name == "StartProcess" ||
					syscallAliases[identifier.Name] && (selector.Sel.Name == "Exec" || selector.Sel.Name == "ForkExec" || selector.Sel.Name == "StartProcess") {
					count++
				}
				return true
			})
			if count > 0 {
				relative, relErr := filepath.Rel(doctrineRoot, path)
				if relErr != nil || strings.HasPrefix(relative, "..") {
					return nil, errors.New("production process entrypoint escaped doctrine root")
				}
				actual[filepath.ToSlash(relative)+":"+function.Name.Name] = true
				functionSelectors += count
			}
		}
		if fileSelectors != functionSelectors {
			return nil, errors.New("production process capability exists outside one named function")
		}
	}
	expected := map[string]bool{
		"grant.go:planningGrantGitOutput": true,
		"refresh.go:gitOutput":            true,
	}
	if len(actual) != len(expected) {
		return nil, errors.New("production process entrypoint inventory changed")
	}
	entrypoints := make(map[string]bool)
	for key := range expected {
		if !actual[key] {
			return nil, errors.New("production process entrypoint inventory changed")
		}
		separator := strings.LastIndexByte(key, ':')
		if separator < 0 || separator+1 == len(key) {
			return nil, errors.New("production process entrypoint inventory is malformed")
		}
		entrypoints[key[separator+1:]] = true
	}
	for _, path := range paths {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return nil, err
		}
		allowedReferences := make(map[token.Pos]bool)
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if entrypoints[function.Name.Name] {
				allowedReferences[function.Name.Pos()] = true
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				if !ok || !entrypoints[identifier.Name] {
					return true
				}
				if call.Ellipsis.IsValid() {
					return false
				}
				allowedReferences[identifier.Pos()] = true
				return true
			})
		}
		invalidReference := false
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && entrypoints[identifier.Name] && !allowedReferences[identifier.Pos()] {
				invalidReference = true
			}
			return true
		})
		if invalidReference {
			return nil, errors.New("production process entrypoint has an indirect or variadic caller edge")
		}
	}
	return entrypoints, nil
}

func checkW001DeliveryScannerIgnore(root string, findings *[]Finding) {
	data, err := readRepoFile(root, w001DeliveryScannerIgnorePath)
	expected := strings.Join(w001DeliveryScannerFingerprints, "\n") + "\n"
	if err != nil || string(data) != expected {
		addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_ignore", "scanner ignore must equal the exact signed ten-fingerprint file")
		return
	}
	checkW001DeliveryScannerFingerprintSources(root, w001DeliveryScannerFingerprints, findings)
}

func checkW001DeliveryScannerFingerprintSources(root string, fingerprints []string, findings *[]Finding) {
	if _, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001DeliveryV1PreservedHead+"^{commit}"); err != nil {
		addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_history", "preserved delivery-v1 head must resolve locally")
		return
	}
	seen := make(map[string]bool, len(fingerprints))
	for _, fingerprint := range fingerprints {
		if seen[fingerprint] {
			addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_duplicate", "scanner fingerprint must occur exactly once")
			continue
		}
		seen[fingerprint] = true
		parts := strings.Split(fingerprint, ":")
		if len(parts) != 4 || !sha1Pattern.MatchString(parts[0]) || !safeRelativePath(parts[1]) ||
			parts[2] != "generic-api-key" {
			addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_fingerprint", "scanner fingerprint is not one exact commit:path:rule:line tuple")
			continue
		}
		line, lineErr := strconv.Atoi(parts[3])
		if lineErr != nil || line < 1 {
			addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_fingerprint", "scanner fingerprint line is invalid")
			continue
		}
		if _, ancestryErr := planningGrantGitOutput(root, "merge-base", "--is-ancestor", parts[0], w001DeliveryV1PreservedHead); ancestryErr != nil {
			addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_history", "scanner fingerprint commit must belong to the preserved delivery-v1 history")
			continue
		}
		content, showErr := planningGrantGitOutput(root, "show", parts[0]+":"+parts[1])
		lines := bytes.Split(content, []byte("\n"))
		if showErr != nil || line > len(lines) || len(bytes.TrimSpace(lines[line-1])) == 0 {
			addFinding(findings, w001DeliveryScannerIgnorePath, "public.w001_delivery_scanner_source", "scanner fingerprint must resolve to a nonempty immutable source line")
		}
	}
}

func checkW001DeliveryPriorTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimChronoFixTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV6TagObject {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_prior_tag", "v6 postclaim tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV6TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_prior_tag", "v6 postclaim tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimChronoFixTag, w001PostclaimChronoFixTagMsg)
	if err != nil || target != "c6749bceb7114b16d7941afc7609c158295ccd2b" {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_prior_tag", "v6 postclaim tag target and signature must remain exact")
		return false
	}
	return true
}

func checkW001PostclaimChronology(root string, document strictPlanningGrant, v6IssuedAt time.Time, findings *[]Finding) {
	type phase struct {
		name       string
		issuedPath string
		commitPath string
		tagPath    string
		runPath    string
		commit     string
		tag        string
	}
	for _, item := range []phase{
		{name: "v4", issuedPath: "chronology.v4IssuedAt", commitPath: "chronology.v4CommitAt", tagPath: "chronology.v4TagAt", runPath: "chronology.v4RunStartedAt", commit: w001PostclaimPRFixBase, tag: w001PostclaimHookFixTag},
		{name: "v5", issuedPath: "chronology.v5IssuedAt", commitPath: "chronology.v5CommitAt", tagPath: "chronology.v5TagAt", runPath: "chronology.v5RunStartedAt", commit: w001PostclaimChronoFixBase, tag: w001PostclaimPRFixTag},
	} {
		issued, issueErr := time.Parse(time.RFC3339, scalarValue(document, item.issuedPath))
		committed, commitErr := time.Parse(time.RFC3339, scalarValue(document, item.commitPath))
		tagged, tagErr := time.Parse(time.RFC3339, scalarValue(document, item.tagPath))
		runStarted, runErr := time.Parse(time.RFC3339, scalarValue(document, item.runPath))
		actualCommit, actualCommitErr := planningGrantCommitTime(root, item.commit)
		actualTag, actualTagErr := planningGrantTagTime(root, item.tag)
		if issueErr != nil || commitErr != nil || tagErr != nil || runErr != nil || actualCommitErr != nil || actualTagErr != nil ||
			!actualCommit.Equal(committed) || !actualTag.Equal(tagged) || !committed.Before(issued) || !tagged.Before(issued) || !runStarted.Before(issued) {
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_record", "%s pre-effective publication chronology must match immutable Git objects and server-authoritative evidence", item.name)
		}
	}
	if v6IssuedAt.IsZero() {
		return
	}
	if target, err := planningGrantGitOutput(root, "rev-parse", "--verify", "refs/tags/"+w001PostclaimChronoFixTag+"^{}"); err == nil {
		featureHead := strings.TrimSpace(string(target))
		commitTime, commitErr := planningGrantCommitTime(root, featureHead)
		tagTime, tagErr := planningGrantTagTime(root, w001PostclaimChronoFixTag)
		if commitErr != nil || tagErr != nil || commitTime.Before(v6IssuedAt) || tagTime.Before(v6IssuedAt) {
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_effect", "v6 commit and signed tag must not precede the signed grant effective time")
		}
	} else if head, headErr := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}"); headErr == nil && strings.TrimSpace(string(head)) != w001PostclaimChronoFixBase {
		commitTime, commitErr := planningGrantCommitTime(root, strings.TrimSpace(string(head)))
		if commitErr != nil || commitTime.Before(v6IssuedAt) {
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_effect", "v6 commit must not precede the signed grant effective time")
		}
	}
}

func planningGrantCommitTime(root, commit string) (time.Time, error) {
	output, err := planningGrantGitOutput(root, "show", "-s", "--format=%cI", commit)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(output)))
}

func planningGrantTagTime(root, tag string) (time.Time, error) {
	output, err := planningGrantGitOutput(root, "for-each-ref", "--format=%(taggerdate:iso-strict)", "refs/tags/"+tag)
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return time.Time{}, errors.New("annotated tag time is unavailable")
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(output)))
}

func extractW001ClaimReceiptDigest(evidence []byte) (string, error) {
	const heading = "## Canonical claim receipt\n"
	sectionStart := bytes.Index(evidence, []byte(heading))
	if sectionStart < 0 || bytes.Count(evidence, []byte(heading)) != 1 {
		return "", errors.New("canonical claim receipt section is missing or duplicated")
	}
	section := evidence[sectionStart+len(heading):]
	if next := bytes.Index(section, []byte("\n## ")); next >= 0 {
		section = section[:next]
	}
	const prefix = "- Receipt SHA-256: `"
	start := bytes.Index(section, []byte(prefix))
	if start < 0 || bytes.Count(section, []byte(prefix)) != 1 {
		return "", errors.New("canonical claim receipt digest is missing or duplicated")
	}
	digestStart := start + len(prefix)
	digestEnd := bytes.IndexByte(section[digestStart:], '`')
	if digestEnd < 0 {
		return "", errors.New("canonical claim receipt digest is not closed")
	}
	digest := string(section[digestStart : digestStart+digestEnd])
	if !sha256Pattern.MatchString(digest) {
		return "", errors.New("canonical claim receipt digest must be lowercase SHA-256")
	}
	return digest, nil
}

func scalarValue(document strictPlanningGrant, path string) string {
	values := document.scalars[path]
	if len(values) != 1 {
		return ""
	}
	return values[0]
}

// LoadW001BootstrapGrant returns the exact validated public configuration for
// the one-shot helper. It performs the complete repository grant check first.
func LoadW001BootstrapGrant(repo string) (W001BootstrapGrant, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return W001BootstrapGrant{}, err
	}
	var findings []Finding
	checkWave1PlanningGrant(root, &findings)
	if len(findings) != 0 {
		sortFindings(findings)
		return W001BootstrapGrant{}, fmt.Errorf("W-001 bootstrap grant validation failed: %s: %s", findings[0].Code, findings[0].Message)
	}
	data, err := readRepoFile(root, w001BootstrapGrantPath)
	if err != nil {
		return W001BootstrapGrant{}, err
	}
	document := parseStrictGrant(data, w001BootstrapGrantScalars, w001BootstrapGrantSequences,
		[]string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"})
	grant := W001BootstrapGrant{
		ID: scalarValue(document, "grant.id"), AttemptID: scalarValue(document, "grant.attemptId"), IdempotencyKey: scalarValue(document, "grant.replayRef"),
		Bead: scalarValue(document, "grant.bead"), BaseCommit: scalarValue(document, "grant.baseCommit"), ExpiresAt: scalarValue(document, "grant.expiresAt"),
		WorkingBranch: scalarValue(document, "grant.workingBranch"), Assignee: scalarValue(document, "expected.assignee"),
		AuthorityProjectID:   scalarValue(document, "expected.beadsProject"),
		ExpectedNativeStatus: scalarValue(document, "expected.nativeStatus"), ExpectedLifecycleState: scalarValue(document, "expected.lifecycleState"),
		ExpectedCreatedAt: scalarValue(document, "expected.createdAt"), ExpectedUpdatedAt: scalarValue(document, "expected.updatedAt"),
		ExpectedMetadataSHA256: scalarValue(document, "expected.metadataSHA256"), ExpectedLabelsSHA256: scalarValue(document, "expected.labelsSHA256"),
		ExpectedDependency: scalarValue(document, "expected.dependency"), ExpectedDependencyType: scalarValue(document, "expected.dependencyType"),
		ExpectedDependencyStatus: scalarValue(document, "expected.dependencyNativeStatus"), ExpectedDependencyLifecycle: scalarValue(document, "expected.dependencyLifecycleState"),
		ExpectedDependencySHA256: scalarValue(document, "expected.dependencyDigest"), ExpectedLineageSHA256: scalarValue(document, "expected.lineageDigest"),
		PostNativeStatus: scalarValue(document, "postimage.nativeStatus"), PostLifecycleState: scalarValue(document, "postimage.lifecycleState"),
		PostMetadataSHA256: scalarValue(document, "postimage.metadataSHA256"), PostLabelsSHA256: scalarValue(document, "postimage.labelsSHA256"),
		PostMetadataBase64: scalarValue(document, "postimage.metadataBase64"), RemoveLabel: scalarValue(document, "postimage.removeLabel"), AddLabel: scalarValue(document, "postimage.addLabel"),
		BeadsVersion: scalarValue(document, "toolchain.beadsVersion"), BeadsSourceCommit: scalarValue(document, "toolchain.beadsSourceCommit"),
		BeadsBinarySHA256: scalarValue(document, "toolchain.beadsBinarySHA256"),
		DoltModule:        scalarValue(document, "toolchain.doltModule"), DoltModuleSHA256: scalarValue(document, "toolchain.doltModuleSHA256"),
		GoVersion: scalarValue(document, "toolchain.goVersion"), GoOS: scalarValue(document, "toolchain.goOS"), GoArch: scalarValue(document, "toolchain.goArch"),
		ICUFormula: scalarValue(document, "toolchain.icuFormula"), DoltTestImage: scalarValue(document, "toolchain.doltTestImage"),
		PatchPath: scalarValue(document, "toolchain.patchPath"), PatchSHA256: scalarValue(document, "toolchain.patchSHA256"),
		PatchedBinarySHA256: scalarValue(document, "toolchain.patchedBinarySHA256"), GoBinarySHA256: scalarValue(document, "toolchain.goBinarySHA256"),
		ReviewTag: scalarValue(document, "grant.reviewTag"),
	}
	if _, stateErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); stateErr == nil {
		grant.CorrectionPatchPath = w001PostclaimSecurityPatchPath
		grant.CorrectionPatchSHA256 = w001PostclaimSecurityPatchSHA
		grant.PatchedBinarySHA256 = w001PostclaimSecurityBinarySHA
	}
	if _, stateErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimHookFixPath))); stateErr == nil {
		grant.HookIsolationPatchPath = w001PostclaimHookPatchPath
		grant.HookIsolationPatchSHA256 = w001PostclaimHookPatchSHA
		grant.PatchedBinarySHA256 = w001PostclaimHookBinarySHA
	}
	return grant, nil
}

// LoadW001DeliveryGrant verifies the complete signed delivery contract before
// returning its minimal runtime projection. Runtime callers must still compare
// the returned preimage with a fresh canonical Beads read immediately before
// issuing a development lease.
func LoadW001DeliveryGrant(repo string) (W001DeliveryGrant, error) {
	return loadW001DeliveryGrant(repo, time.Now().UTC())
}

func loadW001DeliveryGrant(repo string, now time.Time) (W001DeliveryGrant, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return W001DeliveryGrant{}, err
	}
	var findings []Finding
	checkW001DeliveryGrant(root, &findings)
	checkW001BootstrapGrant(root, &findings)
	if len(findings) != 0 {
		sortFindings(findings)
		return W001DeliveryGrant{}, fmt.Errorf("W-001 delivery grant validation failed: %s: %s", findings[0].Code, findings[0].Message)
	}
	data, err := readRepoFile(root, w001DeliveryGrantPath)
	if err != nil {
		return W001DeliveryGrant{}, err
	}
	document := parseStrictGrant(data, w001DeliveryGrantScalars, w001DeliveryGrantSequences,
		[]string{"grant", "canonicalPreimage", "publication", "reconciliation", "verification", "integrity"})
	bootstrapData, err := readRepoFile(root, w001BootstrapGrantPath)
	if err != nil {
		return W001DeliveryGrant{}, err
	}
	bootstrapDocument := parseStrictGrant(bootstrapData, w001BootstrapGrantScalars, w001BootstrapGrantSequences,
		[]string{"grant", "expected", "postimage", "toolchain", "verification", "integrity"})
	expiresAt, err := time.Parse(time.RFC3339, scalarValue(document, "grant.expiresAt"))
	if err != nil || !now.Before(expiresAt) {
		return W001DeliveryGrant{}, errors.New("W-001 delivery grant has expired")
	}
	mutationSequence, err := strconv.ParseUint(scalarValue(document, "canonicalPreimage.issueMutationSequence"), 10, 64)
	if err != nil {
		return W001DeliveryGrant{}, errors.New("W-001 delivery mutation sequence is invalid")
	}
	graphRevision, err := strconv.ParseUint(scalarValue(document, "canonicalPreimage.dependencyGraphRevision"), 10, 64)
	if err != nil {
		return W001DeliveryGrant{}, errors.New("W-001 delivery dependency revision is invalid")
	}
	return W001DeliveryGrant{
		ID: scalarValue(document, "grant.id"), Repository: scalarValue(document, "grant.repository"),
		Bead: scalarValue(document, "grant.bead"), Principal: scalarValue(document, "grant.principal"),
		AttemptID: scalarValue(document, "grant.attemptId"), IdempotencyKey: scalarValue(document, "grant.idempotencyKey"),
		BaseCommit: scalarValue(document, "grant.baseCommit"), ExpiresAt: expiresAt,
		ExpectedNativeStatus:    scalarValue(document, "canonicalPreimage.nativeStatus"),
		ExpectedLifecycleState:  scalarValue(document, "canonicalPreimage.lifecycleState"),
		ExpectedAssignee:        scalarValue(document, "canonicalPreimage.assignee"),
		CanonicalClaimAttemptID: scalarValue(bootstrapDocument, "grant.attemptId"),
		WorkVersionGeneration:   scalarValue(document, "canonicalPreimage.workVersionGeneration"),
		WorkVersionIncarnation:  scalarValue(document, "canonicalPreimage.workVersionIncarnation"),
		IssueMutationSequence:   mutationSequence, DependencyGraphRevision: graphRevision,
		CanonicalWorkMutationAllowed: scalarValue(document, "grant.canonicalWorkMutationAllowed") == "true",
		DevelopmentLeaseAllowed:      scalarValue(document, "grant.developmentLeaseAllowed") == "true",
	}, nil
}

// LoadW001BootstrapExecutionAuthorization verifies the separately signed,
// short-lived authorization that can exist only after protected-main review.
func LoadW001BootstrapExecutionAuthorization(repo, path string, grant W001BootstrapGrant) (W001BootstrapExecutionAuthorization, error) {
	root, err := repositoryRoot(repo)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	if path == "" || filepath.Ext(path) != ".json" {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization must be one explicit JSON file")
	}
	for _, candidate := range []string{path, path + ".sig"} {
		info, statErr := os.Lstat(candidate)
		if statErr != nil || !info.Mode().IsRegular() {
			return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization and detached signature must be regular files")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	authorization, err := decodeCanonicalW001ExecutionAuthorization(data)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	signature, err := os.ReadFile(path + ".sig")
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest || verifySSHSig(data, signature, publicKey, w001BootstrapExecutionNamespace) != nil {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization signature is invalid")
	}
	authorization.payloadSHA256 = fileSHA256(data)
	authorization.signatureSHA256 = fileSHA256(signature)
	issuedAt, issueErr := time.Parse(time.RFC3339, authorization.IssuedAt)
	expiresAt, expiryErr := time.Parse(time.RFC3339, authorization.ExpiresAt)
	now := time.Now().UTC()
	if issueErr != nil || expiryErr != nil || expiresAt.Sub(issuedAt) <= 0 || expiresAt.Sub(issuedAt) > time.Hour || now.Before(issuedAt.Add(-5*time.Minute)) || !now.Before(expiresAt) {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization is outside its signed one-hour validity window")
	}
	if authorization.SchemaVersion != 1 || authorization.Kind != "MARS3W001BootstrapExecutionAuthorization" || authorization.Classification != "PUBLIC" ||
		authorization.GrantID != grant.ID || authorization.Repository != planningGrantRepository || authorization.AttemptID != grant.AttemptID ||
		authorization.IdempotencyKey != grant.IdempotencyKey || authorization.Bead != grant.Bead || authorization.AuthorityProjectID != grant.AuthorityProjectID ||
		authorization.ReviewTag != grant.ReviewTag || authorization.PullRequest != 6 || authorization.ProtectedMainCheckRun <= 0 ||
		authorization.QADisposition != "accepted" || authorization.SecurityDisposition != "accepted" ||
		authorization.QAReviewedCommit != authorization.ReviewedFeatureCommit || authorization.SecurityReviewedCommit != authorization.ReviewedFeatureCommit ||
		authorization.PatchedBinarySHA256 != grant.PatchedBinarySHA256 || authorization.ExpectedMetadataSHA256 != grant.ExpectedMetadataSHA256 ||
		!sha256Pattern.MatchString(authorization.WorkspaceInstanceSHA256) ||
		authorization.AllowedEffect != "execute-one-expected-preimage-W-001-CAS-claim" ||
		!sha1Pattern.MatchString(authorization.MergedCommit) || !sha1Pattern.MatchString(authorization.ReviewedFeatureCommit) ||
		!sha1Pattern.MatchString(authorization.MergedTree) || authorization.MergedCommit == authorization.ReviewedFeatureCommit {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization does not match the exact post-review claim contract")
	}
	return authorization, nil
}

func decodeCanonicalW001ExecutionAuthorization(data []byte) (W001BootstrapExecutionAuthorization, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var authorization W001BootstrapExecutionAuthorization
	if err := decoder.Decode(&authorization); err != nil {
		return W001BootstrapExecutionAuthorization{}, fmt.Errorf("decode execution authorization: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization must contain exactly one JSON object")
	}
	canonical, err := json.Marshal(authorization)
	if err != nil {
		return W001BootstrapExecutionAuthorization{}, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(data, canonical) {
		return W001BootstrapExecutionAuthorization{}, errors.New("execution authorization must use exact canonical JSON with one trailing newline")
	}
	return authorization, nil
}

type planningGrantCheckoutKind int

const (
	planningGrantLocalBranch planningGrantCheckoutKind = iota + 1
	planningGrantPullRequestMerge
	planningGrantMainSquash
)

type planningGrantCheckout struct {
	kind         planningGrantCheckoutKind
	expectedHead string
	firstParent  string
	secondParent string
	tagTarget    string
}

type planningGrantCommit struct {
	id      string
	parents []string
}

type planningGrantGitHubEvent struct {
	After      string `json:"after"`
	Before     string `json:"before"`
	Number     int    `json:"number"`
	Ref        string `json:"ref"`
	HeadCommit *struct {
		ID string `json:"id"`
	} `json:"head_commit"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest *struct {
		MergeCommitSHA string `json:"merge_commit_sha"`
		Base           struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

func checkWave1PlanningGrantGitDiff(root string, findings *[]Finding) {
	metadata, err := os.Lstat(filepath.Join(root, ".git"))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "repository Git metadata cannot be inspected")
		return
	}
	if metadata.Mode()&os.ModeSymlink != 0 || (!metadata.IsDir() && !metadata.Mode().IsRegular()) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "repository Git metadata is not a regular directory or worktree pointer")
		return
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001BootstrapGrantPath))); err == nil {
		checkW001BootstrapGrantGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_state", "W-001 bootstrap grant state cannot be established")
		return
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1DirectMainGrantPath))); err == nil {
		checkWave1DirectMainTransitionGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_state", "direct-main transition grant state cannot be established")
		return
	}

	plan, err := readRepoFile(root, canonicalActivePlan)
	if err != nil {
		addFinding(findings, canonicalActivePlan, "public.planning_grant_diff_phase", "active plan phase cannot be established for a Git checkout")
		return
	}
	phaseMatches := planPhaseLine.FindAllSubmatch(plan, -1)
	if len(phaseMatches) != 1 {
		addFinding(findings, canonicalActivePlan, "public.planning_grant_diff_phase", "active plan must declare exactly one phase before grant diff validation")
		return
	}
	if string(phaseMatches[0][1]) != planPhaseContractPublication {
		addFinding(findings, canonicalActivePlan, "public.planning_grant_transition_authority", "delivery requires a separately signed W-001 bootstrap grant and canonical claim evidence before contract-publication enforcement can end")
		return
	}

	topLevel, err := planningGrantGitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil || !samePlanningGrantRepositoryRoot(root, strings.TrimSpace(string(topLevel))) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "Git metadata must resolve to the audited repository root")
		return
	}
	resolvedBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PlanningGrantBase+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolvedBase)) != wave1PlanningGrantBase {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_base", "the exact signed planning-grant base commit must resolve locally")
		return
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1PlanningGrantBase, "HEAD"); err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_ancestry", "the exact signed planning-grant base commit must be an ancestor of HEAD")
		return
	}
	if !checkWave1PriorPublicationTag(root, findings) {
		return
	}

	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(headOutput))) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_git", "HEAD must resolve to one exact commit")
		return
	}
	head := strings.TrimSpace(string(headOutput))
	checkout, ok := planningGrantCheckoutState(root, head, findings)
	if !ok {
		return
	}
	historyEnd := head
	if checkout.kind != planningGrantLocalBranch {
		tagTarget, tagOK := planningGrantPublicationTagTarget(root, checkout, head, findings)
		if !tagOK {
			return
		}
		checkout.tagTarget = tagTarget
		if checkout.kind == planningGrantMainSquash {
			historyEnd = tagTarget
		}
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1V3AddendumBase, historyEnd); err != nil {
		addFinding(findings, wave1V3AddendumPath, "public.ci_recovery_v3_addendum_ancestry", "the exact v3 CI recovery base must be an ancestor of the effective publication history")
		return
	}
	commits, err := planningGrantCommitRange(root, historyEnd)
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_history", "contract-publication commit ancestry cannot be enumerated")
		return
	}
	if !checkPlanningGrantCommitTopology(root, checkout, head, commits, findings) {
		return
	}

	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, err := openSSHPublicKeyFingerprint(publicKey); err != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_signature", "contract-publication commits cannot be verified with the pinned genesis signer")
	}

	legacyAuthorized := make(map[string]bool, len(wave1PlanningGrantSequences["grant.authorizedPaths"])+len(wave1DispositionSequences["disposition.authorizedPaths"]))
	for _, path := range wave1PlanningGrantSequences["grant.authorizedPaths"] {
		legacyAuthorized[path] = true
	}
	for _, path := range wave1DispositionSequences["disposition.authorizedPaths"] {
		legacyAuthorized[path] = true
	}
	addendumAuthorized := make(map[string]bool, len(wave1AddendumSequences["addendum.authorizedPaths"]))
	for _, path := range wave1AddendumSequences["addendum.authorizedPaths"] {
		addendumAuthorized[path] = true
	}
	v3AddendumAuthorized := make(map[string]bool, len(wave1V3AddendumSequences["addendum.authorizedPaths"]))
	for _, path := range wave1V3AddendumSequences["addendum.authorizedPaths"] {
		v3AddendumAuthorized[path] = true
	}

	signatureFailure := false
	for _, commit := range commits {
		if len(commit.parents) == 1 && keyValid {
			object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
				signatureFailure = true
			}
		}
		if len(commit.parents) != 1 {
			continue
		}
		paths, err := planningGrantGitOutput(root, "diff-tree", "--root", "-m", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id)
		if err != nil {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "per-commit contract-publication paths cannot be enumerated")
			return
		}
		normalized, err := normalizedPlanningGrantGitPaths(paths)
		if err != nil {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "per-commit contract-publication paths are not canonical safe repository-relative names")
			return
		}
		authorized := v3AddendumAuthorized
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, wave1AddendumBase); err == nil {
			authorized = legacyAuthorized
		} else if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, wave1V3AddendumBase); err == nil {
			authorized = addendumAuthorized
		} else if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1V3AddendumBase, commit.id); err != nil {
			addFinding(findings, wave1AddendumPath, "public.ci_recovery_addendum_ancestry", "contract-publication history diverges from the exact CI recovery base")
			return
		}
		if !planningGrantPathsAllowed(normalized, authorized) {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_scope", "a contract-publication commit includes a path outside its signed authorization phase")
			return
		}
	}
	if signatureFailure {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_signature", "every non-merge contract-publication commit must carry a valid pinned SSH signature")
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "current index and worktree paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "untracked contract-publication paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	if err != nil {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_paths", "contract-publication paths are not canonical safe repository-relative names")
		return
	}
	if !planningGrantPathsAllowed(paths, v3AddendumAuthorized) {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_diff_scope", "current v3 CI recovery changes include a path outside the signed v3 addendum")
		return
	}
}

func checkW001PostclaimGrantGitDiff(root string, findings *[]Finding) {
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryGrantPath))); err == nil {
		checkW001DeliveryGrantGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_state", "W-001 delivery-grant Git state cannot be established")
		return
	}
	ciFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimCIFixPath))); err == nil {
		ciFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_state", "postclaim CI-stabilization Git state cannot be established")
		return
	}
	securityFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimSecurityFixPath))); err == nil {
		securityFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_state", "postclaim Security-correction Git state cannot be established")
		return
	}
	if securityFixActive && !ciFixActive {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_ancestry", "Security correction requires the preserved v2 CI stabilization")
		return
	}
	hookFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimHookFixPath))); err == nil {
		hookFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_state", "postclaim hook-isolation Git state cannot be established")
		return
	}
	if hookFixActive && !securityFixActive {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_ancestry", "hook isolation requires the preserved v3 Security correction")
		return
	}
	prFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimPRFixPath))); err == nil {
		prFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_state", "postclaim publication-binding Git state cannot be established")
		return
	}
	if prFixActive && !hookFixActive {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_ancestry", "publication binding requires the preserved v4 hook isolation")
		return
	}
	chronoFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimChronoFixPath))); err == nil {
		chronoFixActive = true
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_state", "postclaim chronology-correction Git state cannot be established")
		return
	}
	if chronoFixActive && !prFixActive {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_ancestry", "chronology correction requires the preserved v5 publication binding")
		return
	}
	base, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimBase+"^{commit}")
	if err != nil || strings.TrimSpace(string(base)) != w001PostclaimBase {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_base", "exact accepted helper squash must resolve locally")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimBase+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != w001PostclaimBaseTree {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_base", "accepted helper squash tree must match the signed base tree")
		return
	}
	if ciFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimCIFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimCIFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimCIFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimCIFixBaseTree {
			addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_base", "CI stabilization must descend from the exact preserved failed head and tree")
			return
		}
		if !checkW001PostclaimPriorReviewTag(root, findings) {
			return
		}
	}
	if securityFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimSecurityFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimSecurityFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimSecurityFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimSecurityFixTree {
			addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_base", "Security correction must descend from the exact accepted v2 head and tree")
			return
		}
		if !checkW001PostclaimPriorV2ReviewTag(root, findings) {
			return
		}
	}
	if hookFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimHookFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimHookFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimHookFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimHookFixTree {
			addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_base", "hook isolation must descend from the exact reviewed v3 head and tree")
			return
		}
		if !checkW001PostclaimPriorV3ReviewTag(root, findings) {
			return
		}
	}
	if prFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimPRFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimPRFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimPRFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimPRFixTree {
			addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_base", "publication binding must descend from the exact reviewed v4 head and tree")
			return
		}
		if !checkW001PostclaimPriorV4ReviewTag(root, findings) {
			return
		}
	}
	if chronoFixActive {
		fixBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimChronoFixBase+"^{commit}")
		fixTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001PostclaimChronoFixBase+"^{tree}")
		if err != nil || treeErr != nil || strings.TrimSpace(string(fixBase)) != w001PostclaimChronoFixBase || strings.TrimSpace(string(fixTree)) != w001PostclaimChronoFixTree {
			addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_base", "chronology correction must descend from the exact reviewed v5 head and tree")
			return
		}
		if !checkW001PostclaimPriorV5ReviewTag(root, findings) {
			return
		}
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := strings.TrimSpace(string(headOutput))
	if err != nil || !sha1Pattern.MatchString(head) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_git", "HEAD must resolve to one exact commit")
		return
	}
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	featureHead := head
	requireTag := false
	mainTreeCheck := false
	switch {
	case branchErr == nil && branch == w001PostclaimBranch && os.Getenv("GITHUB_ACTIONS") != "true":
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimBase, head); err != nil {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_ancestry", "local reconciliation must descend from the exact accepted helper squash")
			return
		}
	case branchErr == nil && branch == "main" && os.Getenv("GITHUB_ACTIONS") != "true":
		requireTag, mainTreeCheck = true, true
	case os.Getenv("GITHUB_ACTIONS") == "true":
		featureHead, requireTag, mainTreeCheck = w001PostclaimGitHubCheckout(root, head, branch, findings)
		if featureHead == "" {
			return
		}
	default:
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_branch", "postclaim reconciliation requires its signed branch or accepted main")
		return
	}

	if requireTag {
		expected := featureHead
		if mainTreeCheck {
			expected = ""
		}
		reviewTag := w001PostclaimReviewTag
		reviewTagMessage := w001PostclaimReviewTagMessage
		if ciFixActive {
			reviewTag = w001PostclaimCIFixReviewTag
			reviewTagMessage = w001PostclaimCIFixTagMessage
		}
		if securityFixActive {
			reviewTag = w001PostclaimSecurityFixTag
			reviewTagMessage = w001PostclaimSecurityFixTagMsg
		}
		if hookFixActive {
			reviewTag = w001PostclaimHookFixTag
			reviewTagMessage = w001PostclaimHookFixTagMsg
		}
		if prFixActive {
			reviewTag = w001PostclaimPRFixTag
			reviewTagMessage = w001PostclaimPRFixTagMsg
		}
		if chronoFixActive {
			reviewTag = w001PostclaimChronoFixTag
			reviewTagMessage = w001PostclaimChronoFixTagMsg
		}
		target, ok := checkW001PostclaimReviewTag(root, expected, reviewTag, reviewTagMessage, findings)
		if !ok {
			return
		}
		featureHead = target
	}
	if mainTreeCheck {
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != w001PostclaimBase {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_main_topology", "accepted reconciliation must be one squash commit over the signed base")
			return
		}
		mainTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", featureHead+"^{tree}")
		if strings.TrimSpace(string(mainTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_main_tree", "accepted reconciliation tree must equal the signed reviewed feature tree")
			return
		}
	}

	if featureHead != w001PostclaimBase {
		commits, err := planningGrantCommitRangeFrom(root, w001PostclaimBase, featureHead)
		if err != nil || len(commits) == 0 {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_history", "reconciliation feature history must be a nonempty linear chain")
			return
		}
		publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
		if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
			addFinding(findings, wave1PlanningGrantKey, "public.w001_postclaim_commit_signature", "reconciliation commits require the pinned genesis signer")
			return
		}
		previous := w001PostclaimBase
		v1Authorized := w001PostclaimPathSet()
		fixAuthorized := w001PostclaimCIFixPathSet()
		securityAuthorized := w001PostclaimSecurityFixPathSet()
		hookAuthorized := w001PostclaimHookFixPathSet()
		prAuthorized := w001PostclaimPRFixPathSet()
		chronoAuthorized := w001PostclaimChronoFixPathSet()
		for _, commit := range commits {
			if len(commit.parents) != 1 || commit.parents[0] != previous {
				addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_topology", "reconciliation feature history must be a contiguous one-parent chain")
				return
			}
			authorized := v1Authorized
			if ciFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimCIFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimCIFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_ancestry", "correction history diverges from the preserved failed head")
						return
					}
					authorized = fixAuthorized
				}
			}
			if securityFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimSecurityFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimSecurityFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_ancestry", "Security-correction history diverges from the preserved v2 head")
						return
					}
					authorized = securityAuthorized
				}
			}
			if hookFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimHookFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimHookFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_ancestry", "hook-isolation history diverges from the preserved v3 head")
						return
					}
					authorized = hookAuthorized
				}
			}
			if prFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimPRFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimPRFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_ancestry", "publication-binding history diverges from the preserved v4 head")
						return
					}
					authorized = prAuthorized
				}
			}
			if chronoFixActive {
				if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001PostclaimChronoFixBase); err != nil {
					if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimChronoFixBase, commit.id); err != nil {
						addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_ancestry", "chronology-correction history diverges from the preserved v5 head")
						return
					}
					authorized = chronoAuthorized
				}
			}
			paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
			normalized, normalizeErr := normalizedPlanningGrantGitPaths(paths)
			if err != nil || normalizeErr != nil || !planningGrantPathsAllowed(normalized, authorized) {
				addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "a reconciliation commit includes a path outside its signed scope")
				return
			}
			object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
				addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_commit_signature", "every reconciliation feature commit must carry the pinned SSH signature")
				return
			}
			previous = commit.id
		}
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "current tracked reconciliation paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "current untracked reconciliation paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	currentAuthorized := w001PostclaimPathSet()
	if ciFixActive {
		currentAuthorized = w001PostclaimCIFixPathSet()
	}
	if securityFixActive {
		currentAuthorized = w001PostclaimSecurityFixPathSet()
	}
	if hookFixActive {
		currentAuthorized = w001PostclaimHookFixPathSet()
	}
	if prFixActive {
		currentAuthorized = w001PostclaimPRFixPathSet()
	}
	if chronoFixActive {
		currentAuthorized = w001PostclaimChronoFixPathSet()
	}
	if err != nil || !planningGrantPathsAllowed(paths, currentAuthorized) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_scope", "current changes include a path outside the signed reconciliation scope")
	}
}

func w001PostclaimPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimGrantSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimGrantSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimCIFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimCIFixSequences["addendum.authorizedPaths"]))
	for _, path := range w001PostclaimCIFixSequences["addendum.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimSecurityFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimSecurityFixSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimSecurityFixSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimHookFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimHookFixSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimHookFixSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimPRFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimPRFixSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimPRFixSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001PostclaimChronoFixPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001PostclaimChronoFixSequences["grant.authorizedPaths"]))
	for _, path := range w001PostclaimChronoFixSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	return authorized
}

func checkW001LifecycleCompletionGitDiff(root string, findings *[]Finding) {
	topLevel, err := planningGrantGitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil || !samePlanningGrantRepositoryRoot(root, strings.TrimSpace(string(topLevel))) {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_git", "Git metadata must resolve to the audited repository root")
		return
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleBase+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001LifecycleBase+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001LifecycleBase || strings.TrimSpace(string(baseTree)) != w001LifecycleBaseTree {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_base", "exact lifecycle-completion base commit and tree must resolve locally")
		return
	}
	if !checkW001LifecyclePriorTag(root, findings) {
		return
	}
	correctionActive := false
	v7Active := false
	v8Active := false
	v9Active := false
	v10Active := false
	v11Active := false
	v12Active := false
	v13Active := false
	v14Active := false
	v15Active := false
	v16Active := false
	v17Active := false
	if _, correctionErr := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionPath))); correctionErr == nil {
		correctionActive = true
		if !checkW001LifecycleV5Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(correctionErr) {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_state", "lifecycle correction Git state cannot be established")
		return
	}
	if _, v7Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV7Path))); v7Err == nil {
		v7Active = true
		if !checkW001LifecycleV6Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v7Err) {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_state", "v7 lifecycle correction Git state cannot be established")
		return
	}
	if _, v8Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV8Path))); v8Err == nil {
		v8Active = true
		if !checkW001LifecycleV7Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v8Err) {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_state", "v8 lifecycle correction Git state cannot be established")
		return
	}
	if _, v9Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCorrectionV9Path))); v9Err == nil {
		v9Active = true
		if !checkW001LifecycleV8Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v9Err) {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_state", "v9 lifecycle correction Git state cannot be established")
		return
	}
	if _, v10Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleStabilizationV10Path))); v10Err == nil {
		v10Active = true
		if !checkW001LifecycleV9Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v10Err) {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_state", "v10 lifecycle CI stabilization Git state cannot be established")
		return
	}
	if _, v11Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIFencingV11Path))); v11Err == nil {
		v11Active = true
		if !checkW001LifecycleV10Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v11Err) {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_state", "v11 lifecycle CI fencing Git state cannot be established")
		return
	}
	if _, v12Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV12Path))); v12Err == nil {
		v12Active = true
		if !checkW001LifecycleV11Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v12Err) {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_state", "v12 lifecycle CI hardening Git state cannot be established")
		return
	}
	if _, v13Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV13Path))); v13Err == nil {
		v13Active = true
		if !checkW001LifecycleV12Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v13Err) {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_state", "v13 lifecycle CI hardening Git state cannot be established")
		return
	}
	if _, v14Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV14Path))); v14Err == nil {
		v14Active = true
		if !checkW001LifecycleV13Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v14Err) {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_state", "v14 lifecycle CI hardening Git state cannot be established")
		return
	}
	if _, v15Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV15Path))); v15Err == nil {
		v15Active = true
		if !checkW001LifecycleV14Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v15Err) {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_state", "v15 lifecycle CI hardening Git state cannot be established")
		return
	}
	if _, v16Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV16Path))); v16Err == nil {
		v16Active = true
		if !checkW001LifecycleV15Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v16Err) {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_state", "v16 lifecycle CI hardening Git state cannot be established")
		return
	}
	if _, v17Err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleCIHardeningV17Path))); v17Err == nil {
		v17Active = true
		if !checkW001LifecycleV16Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(v17Err) {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_state", "v17 lifecycle CI hardening Git state cannot be established")
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := strings.TrimSpace(string(headOutput))
	if err != nil || !sha1Pattern.MatchString(head) {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_git", "HEAD must resolve to one exact commit")
		return
	}
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	featureHead := head
	requireTag := false
	mainTreeCheck := false
	switch {
	case branchErr == nil && branch == w001LifecycleBranch && os.Getenv("GITHUB_ACTIONS") != "true":
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleBase, head); err != nil {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_ancestry", "local lifecycle completion must descend from the exact accepted core squash")
			return
		}
	case branchErr == nil && branch == "main" && os.Getenv("GITHUB_ACTIONS") != "true":
		requireTag, mainTreeCheck = true, true
	case os.Getenv("GITHUB_ACTIONS") == "true":
		featureHead, requireTag, mainTreeCheck = w001LifecycleGitHubCheckout(root, head, branch, findings)
		if featureHead == "" {
			return
		}
	default:
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_branch", "lifecycle completion requires its signed branch or accepted main")
		return
	}
	if requireTag {
		expected := featureHead
		if mainTreeCheck {
			expected = ""
		}
		reviewTag, reviewMessage := w001LifecycleReviewTag, w001LifecycleReviewTagMessage
		if correctionActive {
			reviewTag, reviewMessage = w001LifecycleCorrectionReviewTag, w001LifecycleCorrectionTagMessage
		}
		if v7Active {
			reviewTag, reviewMessage = w001LifecycleCorrectionV7ReviewTag, w001LifecycleCorrectionV7TagMessage
		}
		if v8Active {
			reviewTag, reviewMessage = w001LifecycleCorrectionV8ReviewTag, w001LifecycleCorrectionV8TagMessage
		}
		if v9Active {
			reviewTag, reviewMessage = w001LifecycleCorrectionV9ReviewTag, w001LifecycleCorrectionV9TagMessage
		}
		if v10Active {
			reviewTag, reviewMessage = w001LifecycleStabilizationV10ReviewTag, w001LifecycleStabilizationV10TagMessage
		}
		if v11Active {
			reviewTag, reviewMessage = w001LifecycleCIFencingV11ReviewTag, w001LifecycleCIFencingV11TagMessage
		}
		if v12Active {
			reviewTag, reviewMessage = w001LifecycleCIHardeningV12ReviewTag, w001LifecycleCIHardeningV12TagMessage
		}
		if v13Active {
			reviewTag, reviewMessage = w001LifecycleCIHardeningV13ReviewTag, w001LifecycleCIHardeningV13TagMessage
		}
		if v14Active {
			reviewTag, reviewMessage = w001LifecycleCIHardeningV14ReviewTag, w001LifecycleCIHardeningV14TagMessage
		}
		if v15Active {
			reviewTag, reviewMessage = w001LifecycleCIHardeningV15ReviewTag, w001LifecycleCIHardeningV15TagMessage
		}
		if v16Active {
			reviewTag, reviewMessage = w001LifecycleCIHardeningV16ReviewTag, w001LifecycleCIHardeningV16TagMessage
		}
		if v17Active {
			reviewTag, reviewMessage = w001LifecycleCIHardeningV17ReviewTag, w001LifecycleCIHardeningV17TagMessage
		}
		target, ok := checkW001DeliveryReviewTag(root, expected, reviewTag, reviewMessage, findings)
		if !ok {
			return
		}
		featureHead = target
	}
	if mainTreeCheck {
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != w001LifecycleBase {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_main_topology", "accepted lifecycle completion must be one squash commit over the signed base")
			return
		}
		mainTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", featureHead+"^{tree}")
		if strings.TrimSpace(string(mainTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_main_tree", "accepted lifecycle completion tree must equal the signed reviewed feature tree")
			return
		}
	}
	if featureHead != w001LifecycleBase {
		v5End := featureHead
		v6End := featureHead
		v7End := featureHead
		v8End := featureHead
		v9End := featureHead
		v10End := featureHead
		v11End := featureHead
		v12End := featureHead
		v13End := featureHead
		v14End := featureHead
		v15End := featureHead
		v16End := featureHead
		if correctionActive {
			v5End = w001LifecycleCorrectionBase
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCorrectionBase, featureHead); err != nil {
				addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_ancestry", "lifecycle correction must descend from the exact immutable v5 head")
				return
			}
		}
		if v7Active {
			v6End = w001LifecycleCorrectionV7Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCorrectionV7Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_ancestry", "v7 lifecycle correction must descend from the exact immutable v6 head")
				return
			}
		}
		if v8Active {
			v7End = w001LifecycleCorrectionV8Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCorrectionV8Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_ancestry", "v8 lifecycle correction must descend from the exact immutable v7 head")
				return
			}
		}
		if v9Active {
			v8End = w001LifecycleCorrectionV9Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCorrectionV9Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_ancestry", "v9 lifecycle correction must descend from the exact immutable v8 head")
				return
			}
		}
		if v10Active {
			v9End = w001LifecycleStabilizationV10Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleStabilizationV10Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_ancestry", "v10 lifecycle CI stabilization must descend from the exact immutable v9 head")
				return
			}
		}
		if v11Active {
			v10End = w001LifecycleCIFencingV11Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIFencingV11Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_ancestry", "v11 lifecycle CI fencing must descend from the exact immutable v10 head")
				return
			}
		}
		if v12Active {
			v11End = w001LifecycleCIHardeningV12Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIHardeningV12Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_ancestry", "v12 lifecycle CI hardening must descend from the exact immutable v11 head")
				return
			}
		}
		if v13Active {
			v12End = w001LifecycleCIHardeningV13Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIHardeningV13Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_ancestry", "v13 lifecycle CI hardening must descend from the exact immutable v12 head")
				return
			}
		}
		if v14Active {
			v13End = w001LifecycleCIHardeningV14Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIHardeningV14Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_ancestry", "v14 lifecycle CI hardening must descend from the exact immutable v13 head")
				return
			}
		}
		if v15Active {
			v14End = w001LifecycleCIHardeningV15Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIHardeningV15Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_ancestry", "v15 lifecycle CI hardening must descend from the exact immutable v14 head")
				return
			}
		}
		if v16Active {
			v15End = w001LifecycleCIHardeningV16Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIHardeningV16Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_ancestry", "v16 lifecycle CI hardening must descend from the exact immutable v15 head")
				return
			}
		}
		if v17Active {
			v16End = w001LifecycleCIHardeningV17Base
			if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001LifecycleCIHardeningV17Base, featureHead); err != nil {
				addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_ancestry", "v17 lifecycle CI hardening must descend from the exact immutable v16 head")
				return
			}
		}
		if !checkW001LifecycleCommitRange(root, w001LifecycleBase, v5End, "2026-08-27T12:05:00Z", "v5", findings) {
			return
		}
		if correctionActive && !checkW001LifecycleCommitRange(root, w001LifecycleCorrectionBase, v6End, "2026-08-27T13:50:00Z", "v6", findings) {
			return
		}
		if v7Active && !checkW001LifecycleCommitRange(root, w001LifecycleCorrectionV7Base, v7End, "2026-08-27T15:07:00Z", "v7", findings) {
			return
		}
		if v8Active && !checkW001LifecycleCommitRange(root, w001LifecycleCorrectionV8Base, v8End, "2026-08-27T16:29:00Z", "v8", findings) {
			return
		}
		if v9Active && !checkW001LifecycleCommitRange(root, w001LifecycleCorrectionV9Base, v9End, "2026-08-27T17:29:00Z", "v9", findings) {
			return
		}
		if v10Active && !checkW001LifecycleCommitRange(root, w001LifecycleStabilizationV10Base, v10End, "2026-08-27T18:43:00Z", "v10", findings) {
			return
		}
		if v11Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIFencingV11Base, v11End, "2026-08-27T19:05:55Z", "v11", findings) {
			return
		}
		if v12Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIHardeningV12Base, v12End, "2026-08-27T19:32:00Z", "v12", findings) {
			return
		}
		if v13Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIHardeningV13Base, v13End, "2026-08-27T20:00:30Z", "v13", findings) {
			return
		}
		if v14Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIHardeningV14Base, v14End, "2026-08-27T21:00:28Z", "v14", findings) {
			return
		}
		if v15Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIHardeningV15Base, v15End, "2026-08-28T08:32:45Z", "v15", findings) {
			return
		}
		if v16Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIHardeningV16Base, v16End, "2026-08-28T11:11:48Z", "v16", findings) {
			return
		}
		if v17Active && !checkW001LifecycleCommitRange(root, w001LifecycleCIHardeningV17Base, featureHead, "2026-08-28T20:33:09Z", "v17", findings) {
			return
		}
	}
	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_scope", "current tracked lifecycle-completion paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_scope", "current untracked lifecycle-completion paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	allowed := w001LifecyclePathsAllowed(paths)
	if correctionActive {
		allowed = w001LifecycleCorrectionPathsAllowed(paths)
	}
	if v7Active {
		allowed = w001LifecycleCorrectionV7PathsAllowed(paths)
	}
	if v8Active {
		allowed = w001LifecycleCorrectionV8PathsAllowed(paths)
	}
	if v9Active {
		allowed = w001LifecycleCorrectionV9PathsAllowed(paths)
	}
	if v10Active {
		allowed = w001LifecycleStabilizationV10PathsAllowed(paths)
	}
	if v11Active {
		allowed = w001LifecycleCIFencingV11PathsAllowed(paths)
	}
	if v12Active {
		allowed = w001LifecycleCIHardeningV12PathsAllowed(paths)
	}
	if v13Active {
		allowed = w001LifecycleCIHardeningV13PathsAllowed(paths)
	}
	if v14Active {
		allowed = w001LifecycleCIHardeningV14PathsAllowed(paths)
	}
	if v15Active {
		allowed = w001LifecycleCIHardeningV15PathsAllowed(paths)
	}
	if v16Active {
		allowed = w001LifecycleCIHardeningV16PathsAllowed(paths)
	}
	if v17Active {
		allowed = w001LifecycleCIHardeningV17PathsAllowed(paths)
	}
	if err != nil || !allowed {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_scope", "current changes include a path outside the signed lifecycle-completion scope")
	}
}

func checkW001LifecycleCommitRange(root, base, head, issued, phase string, findings *[]Finding) bool {
	if base == head {
		return true
	}
	commits, err := planningGrantCommitRangeFrom(root, base, head)
	if err != nil || len(commits) == 0 {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_history", "lifecycle history must be a nonempty linear chain")
		return false
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.w001_lifecycle_commit_signature", "lifecycle commits require the pinned genesis signer")
		return false
	}
	issuedAt, parseErr := time.Parse(time.RFC3339, issued)
	if parseErr != nil {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_commit_signature", "lifecycle grant issuance is invalid")
		return false
	}
	previous := base
	for _, commit := range commits {
		if len(commit.parents) != 1 || commit.parents[0] != previous {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_topology", "lifecycle history must be one contiguous one-parent chain")
			return false
		}
		paths, pathErr := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
		normalized, normalizeErr := normalizedPlanningGrantGitPaths(paths)
		allowed := w001LifecyclePathsAllowed(normalized)
		if phase == "v6" {
			allowed = w001LifecycleCorrectionPathsAllowed(normalized)
		} else if phase == "v7" {
			allowed = w001LifecycleCorrectionV7PathsAllowed(normalized)
		} else if phase == "v8" {
			allowed = w001LifecycleCorrectionV8PathsAllowed(normalized)
		} else if phase == "v9" {
			allowed = w001LifecycleCorrectionV9PathsAllowed(normalized)
		} else if phase == "v10" {
			allowed = w001LifecycleStabilizationV10PathsAllowed(normalized)
		} else if phase == "v11" {
			allowed = w001LifecycleCIFencingV11PathsAllowed(normalized)
		} else if phase == "v12" {
			allowed = w001LifecycleCIHardeningV12PathsAllowed(normalized)
		} else if phase == "v13" {
			allowed = w001LifecycleCIHardeningV13PathsAllowed(normalized)
		} else if phase == "v14" {
			allowed = w001LifecycleCIHardeningV14PathsAllowed(normalized)
		} else if phase == "v15" {
			allowed = w001LifecycleCIHardeningV15PathsAllowed(normalized)
		} else if phase == "v16" {
			allowed = w001LifecycleCIHardeningV16PathsAllowed(normalized)
		} else if phase == "v17" {
			allowed = w001LifecycleCIHardeningV17PathsAllowed(normalized)
		}
		object, objectErr := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
		committedAt, timeErr := planningGrantCommitTime(root, commit.id)
		if pathErr != nil || normalizeErr != nil || !allowed {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_scope", "a lifecycle commit includes a path outside its signed phase scope")
			return false
		}
		if objectErr != nil || verifyPlanningGrantCommit(object, publicKey) != nil || timeErr != nil || committedAt.Before(issuedAt) {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_commit_signature", "every lifecycle commit must carry the pinned SSH signature after phase-grant issuance")
			return false
		}
		previous = commit.id
	}
	return true
}

func w001LifecyclePathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleGrantPath: true, w001LifecycleGrantSignature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/features/F-002-work-authority.md": true, "docs/product-specs/work-authority.md": true,
		"docs/evidence/W-001-validation.md": true, "api/authority/v1/types.go": true,
		"internal/doctrine/grant.go": true, "internal/doctrine/grant_test.go": true,
	}
	prefixes := []string{"internal/authority/beads/", "internal/authority/gateway/", "internal/authority/httpapi/", "internal/authority/postgres/", "database/authority/", "cmd/mars3-authority/"}
	for _, path := range paths {
		if exact[path] {
			continue
		}
		allowed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) && len(path) > len(prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func w001LifecycleCorrectionPathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCorrectionPath: true, w001LifecycleCorrectionSignature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/features/F-002-work-authority.md": true, "docs/product-specs/work-authority.md": true,
		"docs/evidence/W-001-validation.md": true, "api/authority/v1/types.go": true,
		"internal/doctrine/grant.go": true, "internal/doctrine/grant_test.go": true,
	}
	prefixes := []string{"internal/authority/beads/", "internal/authority/gateway/", "internal/authority/httpapi/", "internal/authority/postgres/", "database/authority/", "cmd/mars3-authority/"}
	for _, path := range paths {
		if exact[path] {
			continue
		}
		allowed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) && len(path) > len(prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func w001LifecycleCorrectionV7PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCorrectionV7Path: true, w001LifecycleCorrectionV7Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/features/F-002-work-authority.md": true, "docs/product-specs/work-authority.md": true,
		"docs/evidence/W-001-validation.md": true, "api/authority/v1/types.go": true,
		"internal/doctrine/grant.go": true, "internal/doctrine/grant_test.go": true,
	}
	prefixes := []string{"internal/authority/beads/", "internal/authority/gateway/", "internal/authority/httpapi/", "internal/authority/postgres/", "database/authority/", "cmd/mars3-authority/"}
	for _, path := range paths {
		if exact[path] {
			continue
		}
		allowed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) && len(path) > len(prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func w001LifecycleCorrectionV8PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCorrectionV8Path: true, w001LifecycleCorrectionV8Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/features/F-002-work-authority.md": true, "docs/product-specs/work-authority.md": true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if exact[path] || strings.HasPrefix(path, "internal/authority/beads/") && len(path) > len("internal/authority/beads/") {
			continue
		}
		return false
	}
	return true
}

func w001LifecycleCorrectionV9PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCorrectionV9Path: true, w001LifecycleCorrectionV9Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/features/F-002-work-authority.md": true, "docs/product-specs/work-authority.md": true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if exact[path] || strings.HasPrefix(path, "internal/authority/beads/") && len(path) > len("internal/authority/beads/") {
			continue
		}
		return false
	}
	return true
}

func w001LifecycleStabilizationV10PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleStabilizationV10Path: true, w001LifecycleStabilizationV10Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIFencingV11PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIFencingV11Path: true, w001LifecycleCIFencingV11Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIHardeningV12PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIHardeningV12Path: true, w001LifecycleCIHardeningV12Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIHardeningV13PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIHardeningV13Path: true, w001LifecycleCIHardeningV13Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIHardeningV14PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIHardeningV14Path: true, w001LifecycleCIHardeningV14Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIHardeningV15PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIHardeningV15Path: true, w001LifecycleCIHardeningV15Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIHardeningV16PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIHardeningV16Path: true, w001LifecycleCIHardeningV16Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func w001LifecycleCIHardeningV17PathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001LifecycleCIHardeningV17Path: true, w001LifecycleCIHardeningV17Signature: true,
		".harness/manifest.yaml": true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
	}
	for _, path := range paths {
		if !exact[path] {
			return false
		}
	}
	return true
}

func checkW001DeliveryGrantGitDiff(root string, findings *[]Finding) {
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001LifecycleGrantPath))); err == nil {
		checkW001LifecycleCompletionGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_state", "lifecycle-completion Git state cannot be established")
		return
	}
	topLevel, err := planningGrantGitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil || !samePlanningGrantRepositoryRoot(root, strings.TrimSpace(string(topLevel))) {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_git", "Git metadata must resolve to the audited repository root")
		return
	}
	base, baseErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001DeliveryBase+"^{commit}")
	baseTree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", w001DeliveryBase+"^{tree}")
	if baseErr != nil || treeErr != nil || strings.TrimSpace(string(base)) != w001DeliveryBase || strings.TrimSpace(string(baseTree)) != w001DeliveryBaseTree {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_base", "exact delivery base commit and tree must resolve locally")
		return
	}
	if !checkW001DeliveryPriorTag(root, findings) {
		return
	}
	ciFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryCIFixPath))); err == nil {
		ciFixActive = true
		before := len(*findings)
		checkW001DeliveryCIFix(root, findings)
		if len(*findings) != before || !checkW001DeliveryV2Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_state", "delivery CI-correction Git state cannot be established")
		return
	}
	scannerFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001DeliveryScannerFixPath))); err == nil {
		scannerFixActive = true
		before := len(*findings)
		checkW001DeliveryScannerFix(root, findings)
		if len(*findings) != before || !checkW001DeliveryV3Tag(root, findings) {
			return
		}
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_state", "delivery scanner-correction Git state cannot be established")
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := strings.TrimSpace(string(headOutput))
	if err != nil || !sha1Pattern.MatchString(head) {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_git", "HEAD must resolve to one exact commit")
		return
	}
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	featureHead := head
	requireTag := false
	mainTreeCheck := false
	switch {
	case branchErr == nil && branch == w001DeliveryBranch && os.Getenv("GITHUB_ACTIONS") != "true":
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001DeliveryBase, head); err != nil {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_ancestry", "local delivery work must descend from the exact accepted postclaim squash")
			return
		}
	case branchErr == nil && branch == "main" && os.Getenv("GITHUB_ACTIONS") != "true":
		requireTag, mainTreeCheck = true, true
	case os.Getenv("GITHUB_ACTIONS") == "true":
		featureHead, requireTag, mainTreeCheck = w001DeliveryGitHubCheckout(root, head, branch, findings)
		if featureHead == "" {
			return
		}
	default:
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_branch", "delivery requires its signed branch or accepted main")
		return
	}

	if requireTag {
		expected := featureHead
		if mainTreeCheck {
			expected = ""
		}
		reviewTag, reviewTagMessage := w001DeliveryReviewTag, w001DeliveryReviewTagMessage
		if ciFixActive {
			reviewTag, reviewTagMessage = w001DeliveryCIFixReviewTag, w001DeliveryCIFixReviewTagMessage
		}
		if scannerFixActive {
			reviewTag, reviewTagMessage = w001DeliveryScannerFixReviewTag, w001DeliveryScannerFixTagMessage
		}
		target, ok := checkW001DeliveryReviewTag(root, expected, reviewTag, reviewTagMessage, findings)
		if !ok {
			return
		}
		featureHead = target
	}
	if mainTreeCheck {
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != w001DeliveryBase {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_main_topology", "accepted delivery must be one squash commit over the signed base")
			return
		}
		mainTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", featureHead+"^{tree}")
		if strings.TrimSpace(string(mainTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_main_tree", "accepted delivery tree must equal the signed reviewed feature tree")
			return
		}
	}

	if featureHead != w001DeliveryBase {
		commits, err := planningGrantCommitRangeFrom(root, w001DeliveryBase, featureHead)
		if err != nil || len(commits) == 0 {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_history", "delivery feature history must be a nonempty linear chain")
			return
		}
		publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
		if keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
			addFinding(findings, wave1PlanningGrantKey, "public.w001_delivery_commit_signature", "delivery commits require the pinned genesis signer")
			return
		}
		issuedAt, _ := time.Parse(time.RFC3339, "2026-08-27T03:36:00Z")
		previous := w001DeliveryBase
		for _, commit := range commits {
			if len(commit.parents) != 1 || commit.parents[0] != previous {
				addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_topology", "delivery feature history must be a contiguous one-parent chain")
				return
			}
			paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
			normalized, normalizeErr := normalizedPlanningGrantGitPaths(paths)
			pathsAllowed := w001DeliveryPathsAllowed(normalized)
			if ciFixActive {
				if _, ancestryErr := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001DeliveryCIFixBase); ancestryErr != nil {
					pathsAllowed = w001DeliveryCIFixPathsAllowed(normalized)
				}
			}
			if scannerFixActive {
				if _, ancestryErr := planningGrantGitOutput(root, "merge-base", "--is-ancestor", commit.id, w001DeliveryScannerFixBase); ancestryErr != nil {
					pathsAllowed = w001DeliveryScannerFixPathsAllowed(normalized)
				}
			}
			if err != nil || normalizeErr != nil || !pathsAllowed {
				addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_scope", "a delivery commit includes a path outside its signed scope")
				return
			}
			object, objectErr := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			committedAt, timeErr := planningGrantCommitTime(root, commit.id)
			if objectErr != nil || verifyPlanningGrantCommit(object, publicKey) != nil || timeErr != nil || committedAt.Before(issuedAt) {
				addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_commit_signature", "every delivery commit must carry the pinned SSH signature after the grant effective time")
				return
			}
			previous = commit.id
		}
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_scope", "current tracked delivery paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_scope", "current untracked delivery paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	pathsAllowed := w001DeliveryPathsAllowed(paths)
	if ciFixActive {
		pathsAllowed = w001DeliveryCIFixPathsAllowed(paths)
	}
	if scannerFixActive {
		pathsAllowed = w001DeliveryScannerFixPathsAllowed(paths)
	}
	if err != nil || !pathsAllowed {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_scope", "current changes include a path outside the signed delivery scope")
	}
}

func w001DeliveryPathsAllowed(paths []string) bool {
	exact := map[string]bool{
		w001DeliveryGrantPath: true, w001DeliveryGrantSignature: true,
		w001DeliveryCIFixPath: true, w001DeliveryCIFixSignature: true,
		w001DeliveryScannerFixPath: true, w001DeliveryScannerFixSignature: true,
		w001DeliveryScannerIgnorePath: true,
		".harness/manifest.yaml":      true, canonicalActivePlan: true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true, "internal/doctrine/grant_test.go": true,
		"go.mod": true, "go.sum": true, "Makefile": true,
		"NOTICE": true, "THIRD_PARTY_NOTICES": true,
	}
	prefixes := []string{"internal/authority/", "cmd/mars3-authority/", "api/authority/", "database/authority/", "deploy/authority/"}
	for _, path := range paths {
		if exact[path] {
			continue
		}
		allowed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) && len(path) > len(prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func w001DeliveryCIFixPathsAllowed(paths []string) bool {
	allowed := map[string]bool{
		w001DeliveryCIFixPath:               true,
		w001DeliveryCIFixSignature:          true,
		"docs/evidence/W-001-validation.md": true,
		"internal/doctrine/grant.go":        true,
		"internal/doctrine/grant_test.go":   true,
	}
	for _, path := range paths {
		if !allowed[path] {
			return false
		}
	}
	return true
}

func w001DeliveryScannerFixPathsAllowed(paths []string) bool {
	allowed := map[string]bool{
		w001DeliveryScannerIgnorePath:        true,
		w001DeliveryScannerFixPath:           true,
		w001DeliveryScannerFixSignature:      true,
		"docs/evidence/W-001-validation.md":  true,
		"internal/doctrine/grant.go":         true,
		"internal/doctrine/grant_test.go":    true,
		"internal/doctrine/public.go":        true,
		"internal/doctrine/doctrine_test.go": true,
	}
	for _, path := range paths {
		if !allowed[path] {
			return false
		}
	}
	return true
}

func checkW001DeliveryV2Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001DeliveryReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001DeliveryV2TagObject {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_prior_tag", "v2 delivery tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001DeliveryV2TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_prior_tag", "v2 delivery tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTagForIdentity(object, publicKey, w001DeliveryReviewTag, w001DeliveryReviewTagMessage, "engineer@example.com")
	if err != nil || target != w001DeliveryCIFixBase {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_prior_tag", "v2 delivery tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001DeliveryCIFixBaseTree {
		addFinding(findings, w001DeliveryCIFixPath, "public.w001_delivery_ci_prior_tag", "v2 delivery tag tree must remain exact")
		return false
	}
	return true
}

func checkW001DeliveryV3Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001DeliveryCIFixReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001DeliveryV3TagObject {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_prior_tag", "v3 delivery tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001DeliveryV3TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_prior_tag", "v3 delivery tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001DeliveryCIFixReviewTag, w001DeliveryCIFixReviewTagMessage)
	if err != nil || target != w001DeliveryScannerFixBase {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_prior_tag", "v3 delivery tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001DeliveryScannerFixBaseTree {
		addFinding(findings, w001DeliveryScannerFixPath, "public.w001_delivery_scanner_prior_tag", "v3 delivery tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecyclePriorTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001DeliveryScannerFixReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001DeliveryV4TagObject {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_prior_tag", "v4 delivery tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001DeliveryV4TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_prior_tag", "v4 delivery tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001DeliveryScannerFixReviewTag, w001DeliveryScannerFixTagMessage)
	if err != nil || target != "cac4231ddcb69edd298766c5bbe3854c8269fb2a" {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_prior_tag", "v4 delivery tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleBaseTree {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_prior_tag", "v4 delivery tag tree must equal the accepted core squash tree")
		return false
	}
	return true
}

func checkW001LifecycleV5Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV5TagObject {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_prior_tag", "v5 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV5TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_prior_tag", "v5 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleReviewTag, w001LifecycleReviewTagMessage)
	if err != nil || target != w001LifecycleCorrectionBase {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_prior_tag", "v5 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCorrectionBaseTree {
		addFinding(findings, w001LifecycleCorrectionPath, "public.w001_lifecycle_correction_prior_tag", "v5 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV6Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCorrectionReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV6TagObject {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_prior_tag", "v6 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV6TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_prior_tag", "v6 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCorrectionReviewTag, w001LifecycleCorrectionTagMessage)
	if err != nil || target != w001LifecycleCorrectionV7Base {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_prior_tag", "v6 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCorrectionV7BaseTree {
		addFinding(findings, w001LifecycleCorrectionV7Path, "public.w001_lifecycle_correction_v7_prior_tag", "v6 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV7Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCorrectionV7ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV7TagObject {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_prior_tag", "v7 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV7TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_prior_tag", "v7 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCorrectionV7ReviewTag, w001LifecycleCorrectionV7TagMessage)
	if err != nil || target != w001LifecycleCorrectionV8Base {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_prior_tag", "v7 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCorrectionV8BaseTree {
		addFinding(findings, w001LifecycleCorrectionV8Path, "public.w001_lifecycle_correction_v8_prior_tag", "v7 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV8Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCorrectionV8ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV8TagObject {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_prior_tag", "v8 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV8TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_prior_tag", "v8 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCorrectionV8ReviewTag, w001LifecycleCorrectionV8TagMessage)
	if err != nil || target != w001LifecycleCorrectionV9Base {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_prior_tag", "v8 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCorrectionV9BaseTree {
		addFinding(findings, w001LifecycleCorrectionV9Path, "public.w001_lifecycle_correction_v9_prior_tag", "v8 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV9Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCorrectionV9ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV9TagObject {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_prior_tag", "v9 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV9TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_prior_tag", "v9 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCorrectionV9ReviewTag, w001LifecycleCorrectionV9TagMessage)
	if err != nil || target != w001LifecycleStabilizationV10Base {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_prior_tag", "v9 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleStabilizationV10BaseTree {
		addFinding(findings, w001LifecycleStabilizationV10Path, "public.w001_lifecycle_stabilization_v10_prior_tag", "v9 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV10Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleStabilizationV10ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV10TagObject {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_prior_tag", "v10 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV10TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_prior_tag", "v10 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleStabilizationV10ReviewTag, w001LifecycleStabilizationV10TagMessage)
	if err != nil || target != w001LifecycleCIFencingV11Base {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_prior_tag", "v10 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIFencingV11BaseTree {
		addFinding(findings, w001LifecycleCIFencingV11Path, "public.w001_lifecycle_ci_fencing_v11_prior_tag", "v10 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV11Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCIFencingV11ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV11TagObject {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_prior_tag", "v11 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV11TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_prior_tag", "v11 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCIFencingV11ReviewTag, w001LifecycleCIFencingV11TagMessage)
	if err != nil || target != w001LifecycleCIHardeningV12Base {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_prior_tag", "v11 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIHardeningV12BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV12Path, "public.w001_lifecycle_ci_hardening_v12_prior_tag", "v11 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV12Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCIHardeningV12ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV12TagObject {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_prior_tag", "v12 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV12TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_prior_tag", "v12 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCIHardeningV12ReviewTag, w001LifecycleCIHardeningV12TagMessage)
	if err != nil || target != w001LifecycleCIHardeningV13Base {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_prior_tag", "v12 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIHardeningV13BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV13Path, "public.w001_lifecycle_ci_hardening_v13_prior_tag", "v12 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV13Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCIHardeningV13ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV13TagObject {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_prior_tag", "v13 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV13TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_prior_tag", "v13 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCIHardeningV13ReviewTag, w001LifecycleCIHardeningV13TagMessage)
	if err != nil || target != w001LifecycleCIHardeningV14Base {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_prior_tag", "v13 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIHardeningV14BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV14Path, "public.w001_lifecycle_ci_hardening_v14_prior_tag", "v13 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV14Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCIHardeningV14ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV14TagObject {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_prior_tag", "v14 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV14TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_prior_tag", "v14 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCIHardeningV14ReviewTag, w001LifecycleCIHardeningV14TagMessage)
	if err != nil || target != w001LifecycleCIHardeningV15Base {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_prior_tag", "v14 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIHardeningV15BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV15Path, "public.w001_lifecycle_ci_hardening_v15_prior_tag", "v14 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV15Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCIHardeningV15ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV15TagObject {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_prior_tag", "v15 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV15TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_prior_tag", "v15 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCIHardeningV15ReviewTag, w001LifecycleCIHardeningV15TagMessage)
	if err != nil || target != w001LifecycleCIHardeningV16Base {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_prior_tag", "v15 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIHardeningV16BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV16Path, "public.w001_lifecycle_ci_hardening_v16_prior_tag", "v15 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001LifecycleV16Tag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001LifecycleCIHardeningV16ReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001LifecycleV16TagObject {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_prior_tag", "v16 lifecycle tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001LifecycleV16TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_prior_tag", "v16 lifecycle tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001LifecycleCIHardeningV16ReviewTag, w001LifecycleCIHardeningV16TagMessage)
	if err != nil || target != w001LifecycleCIHardeningV17Base {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_prior_tag", "v16 lifecycle tag identity, target, message, and signature must remain exact")
		return false
	}
	tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if treeErr != nil || strings.TrimSpace(string(tree)) != w001LifecycleCIHardeningV17BaseTree {
		addFinding(findings, w001LifecycleCIHardeningV17Path, "public.w001_lifecycle_ci_hardening_v17_prior_tag", "v16 lifecycle tag tree must remain exact")
		return false
	}
	return true
}

func checkW001DeliveryReviewTag(root, expectedFeatureHead, reviewTag, reviewTagMessage string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + reviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(objectID))) {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_tag", "delivery CI requires the signed immutable review tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(objectID)))
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_tag", "delivery review tag cannot be verified with the pinned key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, reviewTag, reviewTagMessage)
	if err != nil {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_tag_identity", "delivery review tag identity, message, and signature must match the signed review contract")
		return "", false
	}
	if expectedFeatureHead != "" && expectedFeatureHead != target {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_tag_target", "delivery review tag must target the immutable feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001DeliveryBase, target); err != nil || target == w001DeliveryBase {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_tag_target", "delivery review tag must attest nonempty bounded delivery history")
		return "", false
	}
	return target, true
}

func w001DeliveryGitHubCheckout(root, head, branch string, findings *[]Finding) (string, bool, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_runner", "delivery GitHub checkout lacks canonical runner identity")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_runner", "delivery GitHub run ID is invalid")
		return "", false, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 || !strings.HasPrefix(workflowRef, workflowPrefix) {
		addFinding(findings, planningGrantWorkflowPath, "public.w001_delivery_workflow", "delivery CI requires the pinned protected workflow")
		return "", false, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_event", "delivery CI event identity is invalid")
		return "", false, false
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) || os.Getenv("GITHUB_HEAD_REF") != w001DeliveryBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.Number <= 0 || event.PullRequest == nil ||
			event.PullRequest.Head.Ref != w001DeliveryBranch || event.PullRequest.Base.Ref != "main" ||
			event.PullRequest.Base.SHA != w001DeliveryBase || !sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_event", "pull-request event does not bind the signed delivery branch and base")
			return "", false, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			addFinding(findings, planningGrantWorkflowPath, "public.w001_delivery_workflow", "pull-request workflow ref is not canonical")
			return "", false, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != w001DeliveryBase || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_pr_topology", "pull-request checkout must be the exact two-parent synthetic merge")
			return "", false, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, featureErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || featureErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_pr_tree", "pull-request synthetic merge tree must equal the reviewed feature tree")
			return "", false, false
		}
		return event.PullRequest.Head.SHA, true, false
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != w001DeliveryBase || event.After != head ||
			event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_event", "protected-main event does not bind the signed delivery base and squash")
			return "", false, false
		}
		return head, true, true
	default:
		addFinding(findings, w001DeliveryGrantPath, "public.w001_delivery_event", "unsupported GitHub event for W-001 delivery")
		return "", false, false
	}
}

func w001LifecycleGitHubCheckout(root, head, branch string, findings *[]Finding) (string, bool, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_runner", "lifecycle-completion checkout lacks canonical runner identity")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_runner", "lifecycle-completion GitHub run ID is invalid")
		return "", false, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 || !strings.HasPrefix(workflowRef, workflowPrefix) {
		addFinding(findings, planningGrantWorkflowPath, "public.w001_lifecycle_workflow", "lifecycle-completion CI requires the pinned protected workflow")
		return "", false, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_event", "lifecycle-completion CI event identity is invalid")
		return "", false, false
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) || os.Getenv("GITHUB_HEAD_REF") != w001LifecycleBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.Number <= 0 || event.PullRequest == nil ||
			event.PullRequest.Head.Ref != w001LifecycleBranch || event.PullRequest.Base.Ref != "main" ||
			event.PullRequest.Base.SHA != w001LifecycleBase || !sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_event", "pull-request event does not bind the signed lifecycle branch and base")
			return "", false, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			addFinding(findings, planningGrantWorkflowPath, "public.w001_lifecycle_workflow", "pull-request workflow ref is not canonical")
			return "", false, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != w001LifecycleBase || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_pr_topology", "pull-request checkout must be the exact two-parent synthetic merge")
			return "", false, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, featureErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || featureErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_pr_tree", "pull-request synthetic merge tree must equal the reviewed feature tree")
			return "", false, false
		}
		return event.PullRequest.Head.SHA, true, false
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != w001LifecycleBase || event.After != head ||
			event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_event", "protected-main event does not bind the signed lifecycle base and squash")
			return "", false, false
		}
		return head, true, true
	default:
		addFinding(findings, w001LifecycleGrantPath, "public.w001_lifecycle_event", "unsupported GitHub event for W-001 lifecycle completion")
		return "", false, false
	}
}

func checkW001PostclaimReviewTag(root, expectedFeatureHead, reviewTag, reviewTagMessage string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + reviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(objectID))) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag", "postclaim CI requires the signed immutable review tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(objectID)))
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag", "postclaim review tag cannot be verified with the pinned key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, reviewTag, reviewTagMessage)
	if err != nil {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag", "postclaim review tag must be an exact pinned-signer tree attestation")
		return "", false
	}
	if expectedFeatureHead != "" && expectedFeatureHead != target {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag_target", "postclaim review tag must target the immutable feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001PostclaimBase, target); err != nil || target == w001PostclaimBase {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_tag_target", "postclaim tag target must preserve nonempty reconciliation history")
		return "", false
	}
	return target, true
}

func checkW001PostclaimPriorReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV1TagObject {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_v1_tag", "v1 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV1TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_v1_tag", "v1 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimReviewTag, w001PostclaimReviewTagMessage)
	if err != nil || target != w001PostclaimCIFixBase {
		addFinding(findings, w001PostclaimCIFixPath, "public.w001_postclaim_ci_v1_tag", "v1 review tag target and signature must remain exact")
		return false
	}
	return true
}

func checkW001PostclaimPriorV2ReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimCIFixReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV2TagObject {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_v2_tag", "v2 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV2TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_v2_tag", "v2 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimCIFixReviewTag, w001PostclaimCIFixTagMessage)
	if err != nil || target != w001PostclaimSecurityFixBase {
		addFinding(findings, w001PostclaimSecurityFixPath, "public.w001_postclaim_security_v2_tag", "v2 review tag target and signature must remain exact")
		return false
	}
	return true
}

func checkW001PostclaimPriorV3ReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimSecurityFixTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV3TagObject {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_v3_tag", "v3 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV3TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_v3_tag", "v3 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimSecurityFixTag, w001PostclaimSecurityFixTagMsg)
	if err != nil || target != w001PostclaimHookFixBase {
		addFinding(findings, w001PostclaimHookFixPath, "public.w001_postclaim_hook_v3_tag", "v3 review tag target and signature must remain exact")
		return false
	}
	return true
}

func checkW001PostclaimPriorV4ReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimHookFixTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV4TagObject {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_v4_tag", "v4 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV4TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_v4_tag", "v4 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimHookFixTag, w001PostclaimHookFixTagMsg)
	if err != nil || target != w001PostclaimPRFixBase {
		addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_v4_tag", "v4 review tag target and signature must remain exact")
		return false
	}
	return true
}

func checkW001PostclaimPriorV5ReviewTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + w001PostclaimPRFixTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(objectID)) != w001PostclaimV5TagObject {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_v5_tag", "v5 review tag object must remain exact and immutable")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", w001PostclaimV5TagObject)
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_v5_tag", "v5 review tag cannot be verified with the pinned key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001PostclaimPRFixTag, w001PostclaimPRFixTagMsg)
	if err != nil || target != w001PostclaimChronoFixBase {
		addFinding(findings, w001PostclaimChronoFixPath, "public.w001_postclaim_chronology_v5_tag", "v5 review tag target and signature must remain exact")
		return false
	}
	return true
}

func w001PostclaimGitHubCheckout(root, head, branch string, findings *[]Finding) (string, bool, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_runner", "postclaim GitHub checkout lacks canonical runner identity")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_runner", "postclaim GitHub run ID is invalid")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_runner", "postclaim GitHub run attempt is invalid")
		return "", false, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 || !strings.HasPrefix(workflowRef, workflowPrefix) {
		addFinding(findings, planningGrantWorkflowPath, "public.w001_postclaim_workflow", "postclaim CI requires the pinned protected workflow")
		return "", false, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "postclaim CI event identity is invalid")
		return "", false, false
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) || os.Getenv("GITHUB_HEAD_REF") != w001PostclaimBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil || event.PullRequest.Head.Ref != w001PostclaimBranch ||
			event.PullRequest.Base.Ref != "main" || event.PullRequest.Base.SHA != w001PostclaimBase ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) || !validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "pull-request event does not bind the signed postclaim branch and base")
			return "", false, false
		}
		if !w001PostclaimPullRequestNumberAllowed(root, event.Number) {
			addFinding(findings, w001PostclaimPRFixPath, "public.w001_postclaim_pr_binding_event", "pull-request event must bind the signed active PR 8 publication vehicle")
			return "", false, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			addFinding(findings, planningGrantWorkflowPath, "public.w001_postclaim_workflow", "pull-request workflow ref is not canonical")
			return "", false, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != w001PostclaimBase || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_pr_topology", "pull-request checkout must be the exact two-parent synthetic merge")
			return "", false, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, featureErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || featureErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_pr_tree", "pull-request synthetic merge tree must equal the reviewed feature tree")
			return "", false, false
		}
		return event.PullRequest.Head.SHA, true, false
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != w001PostclaimBase || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "protected-main event does not bind the signed reconciliation base and squash")
			return "", false, false
		}
		return head, true, true
	default:
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_event", "unsupported GitHub event for postclaim reconciliation")
		return "", false, false
	}
}

func w001PostclaimPullRequestNumberAllowed(root string, number int) bool {
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimPRFixPath))); err == nil {
		return number == w001PostclaimActivePR
	}
	return true
}

func checkW001BootstrapGrantGitDiff(root string, findings *[]Finding) {
	topLevel, err := planningGrantGitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil || !samePlanningGrantRepositoryRoot(root, strings.TrimSpace(string(topLevel))) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_git", "Git metadata must resolve to the audited repository root")
		return
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(w001PostclaimGrantPath))); err == nil {
		checkW001PostclaimGrantGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, w001PostclaimGrantPath, "public.w001_postclaim_state", "postclaim reconciliation Git state cannot be established")
		return
	}
	base, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001BootstrapBase+"^{commit}")
	if err != nil || strings.TrimSpace(string(base)) != w001BootstrapBase {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_base", "the exact signed bootstrap base must resolve locally")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", w001BootstrapBase+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != w001BootstrapBaseTree {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_base", "the exact signed bootstrap base tree must remain unchanged")
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := strings.TrimSpace(string(headOutput))
	if err != nil || !sha1Pattern.MatchString(head) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_git", "HEAD must resolve to one exact commit")
		return
	}
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	featureHead := head
	requireTag := false
	mainTreeCheck := false

	switch {
	case branchErr == nil && branch == w001BootstrapBranch && os.Getenv("GITHUB_ACTIONS") != "true":
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001BootstrapBase, head); err != nil {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_ancestry", "local bootstrap work must descend from the exact signed base")
			return
		}
	case branchErr == nil && branch == "main" && os.Getenv("GITHUB_ACTIONS") != "true":
		requireTag, mainTreeCheck = true, true
	case os.Getenv("GITHUB_ACTIONS") == "true":
		featureHead, requireTag, mainTreeCheck = w001BootstrapGitHubCheckout(root, head, branch, findings)
		if featureHead == "" {
			return
		}
	default:
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_branch", "bootstrap work requires the exact signed branch or accepted main")
		return
	}

	if requireTag {
		tagExpected := featureHead
		if mainTreeCheck {
			tagExpected = ""
		}
		target, ok := checkW001BootstrapReviewTag(root, tagExpected, findings)
		if !ok {
			return
		}
		featureHead = target
	}
	if mainTreeCheck {
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != w001BootstrapBase {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_main_topology", "accepted main must be one squash commit over the signed bootstrap base")
			return
		}
		mainTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, _ := planningGrantGitOutput(root, "rev-parse", "--verify", featureHead+"^{tree}")
		if strings.TrimSpace(string(mainTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_main_tree", "accepted main tree must equal the signed reviewed feature tree")
			return
		}
	}

	if featureHead != w001BootstrapBase {
		commits, err := planningGrantCommitRangeFrom(root, w001BootstrapBase, featureHead)
		if err != nil || len(commits) == 0 {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_history", "bootstrap feature history must be a nonempty linear chain")
			return
		}
		publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
		if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
			addFinding(findings, wave1PlanningGrantKey, "public.w001_bootstrap_commit_signature", "bootstrap commits require the pinned genesis signer")
			return
		}
		previous := w001BootstrapBase
		authorized := w001BootstrapPreclaimPathSet()
		for _, commit := range commits {
			if len(commit.parents) != 1 || commit.parents[0] != previous {
				addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_topology", "bootstrap feature history must be a contiguous one-parent chain")
				return
			}
			object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
			if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
				addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_commit_signature", "every bootstrap feature commit must carry the pinned SSH signature")
				return
			}
			paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
			normalized, normalizeErr := normalizedPlanningGrantGitPaths(paths)
			if err != nil || normalizeErr != nil || !planningGrantPathsAllowed(normalized, authorized) {
				addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "a bootstrap feature commit includes a path outside its signed preclaim scope")
				return
			}
			previous = commit.id
		}
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "current tracked bootstrap paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "current untracked bootstrap paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	if err != nil || !planningGrantPathsAllowed(paths, w001BootstrapPreclaimPathSet()) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_scope", "current changes include a path outside the signed preclaim scope")
	}
}

func w001BootstrapPreclaimPathSet() map[string]bool {
	authorized := make(map[string]bool, len(w001BootstrapGrantSequences["grant.preclaimPaths"]))
	for _, path := range w001BootstrapGrantSequences["grant.preclaimPaths"] {
		authorized[path] = true
	}
	return authorized
}

func w001BootstrapGitHubCheckout(root, head, branch string, findings *[]Finding) (string, bool, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_runner", "GitHub bootstrap checkout lacks canonical runner identity")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_runner", "GitHub bootstrap run ID is invalid")
		return "", false, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_runner", "GitHub bootstrap run attempt is invalid")
		return "", false, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 || !strings.HasPrefix(workflowRef, workflowPrefix) {
		addFinding(findings, planningGrantWorkflowPath, "public.w001_bootstrap_workflow", "bootstrap CI requires the pinned protected workflow")
		return "", false, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "bootstrap CI event identity is invalid")
		return "", false, false
	}
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(os.Getenv("GITHUB_REF")) || os.Getenv("GITHUB_HEAD_REF") != w001BootstrapBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil || event.PullRequest.Head.Ref != w001BootstrapBranch ||
			event.PullRequest.Base.Ref != "main" || event.PullRequest.Base.SHA != w001BootstrapBase ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "pull-request event does not bind the signed branch and base")
			return "", false, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			addFinding(findings, planningGrantWorkflowPath, "public.w001_bootstrap_workflow", "pull-request workflow ref is not canonical")
			return "", false, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != w001BootstrapBase || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_pr_topology", "pull-request checkout must be the exact two-parent synthetic merge")
			return "", false, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		featureTree, featureErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || featureErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(featureTree)) {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_pr_tree", "pull-request synthetic merge tree must equal the reviewed feature tree")
			return "", false, false
		}
		return event.PullRequest.Head.SHA, true, false
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != w001BootstrapBase || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "protected-main event does not bind the signed bootstrap base and squash")
			return "", false, false
		}
		return head, true, true
	default:
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_event", "unsupported GitHub event for bootstrap publication")
		return "", false, false
	}
}

func checkW001BootstrapReviewTag(root, expectedFeatureHead string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + w001BootstrapReviewTag
	objectID, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(objectID))) {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag", "bootstrap CI requires the signed immutable review tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(objectID)))
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || keyErr != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag", "bootstrap review tag cannot be verified with the pinned key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, w001BootstrapReviewTag, w001BootstrapReviewTagMessage)
	if err != nil {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag", "bootstrap review tag must be an exact pinned-signer tree attestation")
		return "", false
	}
	if expectedFeatureHead != "" && expectedFeatureHead != target {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag_target", "review tag must target the immutable feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", w001BootstrapBase, target); err != nil || target == w001BootstrapBase {
		addFinding(findings, w001BootstrapGrantPath, "public.w001_bootstrap_tag_target", "review tag target must preserve nonempty bootstrap feature history")
		return "", false
	}
	return target, true
}

func checkWave1DirectMainTransitionGitDiff(root string, findings *[]Finding) {
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1PRFallbackPath))); err == nil {
		checkWave1PRFallbackGitDiff(root, findings)
		return
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_state", "PR-fallback addendum state cannot be established")
		return
	}
	plan, err := readRepoFile(root, canonicalActivePlan)
	if err != nil {
		addFinding(findings, canonicalActivePlan, "public.direct_main_transition_phase", "active plan phase cannot be established")
		return
	}
	phaseMatches := planPhaseLine.FindAllSubmatch(plan, -1)
	if len(phaseMatches) != 1 || string(phaseMatches[0][1]) != planPhaseContractPublication {
		addFinding(findings, canonicalActivePlan, "public.direct_main_transition_phase", "transition authority permits preparation only while the canonical Bead remains backlog and unclaimed")
		return
	}

	resolvedBase, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PublishedMain+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolvedBase)) != wave1PublishedMain {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_base", "the exact verified publication commit must resolve locally")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PublishedMain+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != wave1PublishedTree {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_base_tree", "the verified publication base tree does not match the signed transition")
		return
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1PublishedMain, "HEAD"); err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_ancestry", "the verified publication commit must be an ancestor of HEAD")
		return
	}
	if !checkWave1PriorPublicationTag(root, findings) || !checkWave1V3PublicationTag(root, findings) {
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(headOutput))) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_head", "HEAD must resolve to one exact commit")
		return
	}
	head := strings.TrimSpace(string(headOutput))
	if !checkWave1DirectMainCheckout(root, head, findings) {
		return
	}

	commits, err := planningGrantCommitRangeFrom(root, wave1PublishedMain, head)
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_history", "post-publication commit ancestry cannot be enumerated")
		return
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.direct_main_transition_commit_signature", "post-publication commits require the pinned genesis signer")
		return
	}
	authorized := make(map[string]bool, len(wave1DirectMainGrantSequences["grant.authorizedPaths"]))
	for _, path := range wave1DirectMainGrantSequences["grant.authorizedPaths"] {
		authorized[path] = true
	}
	previous := wave1PublishedMain
	for _, commit := range commits {
		if len(commit.parents) != 1 || commit.parents[0] != previous {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_topology", "post-publication main history must be a contiguous one-parent chain")
			return
		}
		object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
		if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_commit_signature", "every post-publication commit must carry the pinned SSH signature")
			return
		}
		paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
		if err != nil {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_paths", "post-publication commit paths cannot be enumerated")
			return
		}
		normalized, err := normalizedPlanningGrantGitPaths(paths)
		if err != nil || !planningGrantPathsAllowed(normalized, authorized) {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_scope", "a post-publication commit includes a path outside the signed transition authority")
			return
		}
		previous = commit.id
	}

	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_paths", "current index and worktree paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_paths", "untracked transition paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	if err != nil || !planningGrantPathsAllowed(paths, authorized) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_scope", "current changes include a path outside the signed transition authority")
	}
}

func checkWave1PRFallbackGitDiff(root string, findings *[]Finding) {
	plan, err := readRepoFile(root, canonicalActivePlan)
	if err != nil {
		addFinding(findings, canonicalActivePlan, "public.pr_fallback_phase", "active plan phase cannot be established")
		return
	}
	phaseMatches := planPhaseLine.FindAllSubmatch(plan, -1)
	if len(phaseMatches) != 1 || string(phaseMatches[0][1]) != planPhaseContractPublication {
		addFinding(findings, canonicalActivePlan, "public.pr_fallback_phase", "PR fallback permits transition preparation only while W-001 remains backlog and unclaimed")
		return
	}
	baseTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", wave1PublishedMain+"^{tree}")
	if err != nil || strings.TrimSpace(string(baseTree)) != wave1PublishedTree {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_base", "the exact verified publication base and tree must resolve locally")
		return
	}
	if !checkWave1PriorPublicationTag(root, findings) || !checkWave1V3PublicationTag(root, findings) {
		return
	}
	mainCIFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1MainCIFixPath))); err == nil {
		mainCIFixActive = true
		before := len(*findings)
		checkWave1MainCIFixAddendum(root, findings)
		if len(*findings) != before {
			return
		}
		if !checkWave1PriorTransitionTag(root, findings) {
			return
		}
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_state", "main-CI correction addendum state cannot be established")
		return
	}
	fixtureFixActive := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(wave1CIFixtureFixPath))); err == nil {
		fixtureFixActive = true
		before := len(*findings)
		checkWave1CIFixtureFixAddendum(root, findings)
		if len(*findings) != before {
			return
		}
		if !checkWave1PriorV2TransitionTag(root, findings) {
			return
		}
	} else if !os.IsNotExist(err) {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_state", "fixture-stabilization addendum state cannot be established")
		return
	}
	headOutput, err := planningGrantGitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(headOutput))) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_head", "HEAD must resolve to one exact commit")
		return
	}
	head := strings.TrimSpace(string(headOutput))
	checkout, ok := checkWave1PRFallbackCheckout(root, head, findings)
	if !ok {
		return
	}
	historyEnd := head
	if checkout.kind == planningGrantPullRequestMerge {
		historyEnd = checkout.secondParent
	}
	if checkout.kind != planningGrantLocalBranch {
		tagName := wave1TransitionTag
		tagMessage := wave1TransitionTagMessage
		if mainCIFixActive {
			tagName = wave1SuccessorTransitionTag
			tagMessage = wave1SuccessorTagMessage
		}
		if fixtureFixActive {
			tagName = wave1FinalTransitionTag
			tagMessage = wave1FinalTransitionTagMessage
		}
		expectedTarget := historyEnd
		requireDistinct := false
		if checkout.kind == planningGrantMainSquash {
			expectedTarget = ""
			requireDistinct = true
		}
		tagTarget, tagOK := checkWave1TransitionTag(root, tagName, tagMessage, expectedTarget, head, requireDistinct, findings)
		if !tagOK {
			return
		}
		if checkout.kind == planningGrantMainSquash {
			historyEnd = tagTarget
		}
	}
	commits, err := planningGrantCommitRangeFrom(root, wave1PublishedMain, historyEnd)
	if err != nil || len(commits) == 0 {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_history", "fallback feature history must be a nonempty linear chain")
		return
	}
	publicKey, keyErr := readRepoFile(root, wave1PlanningGrantKey)
	keyValid := keyErr == nil && fileSHA256(publicKey) == genesisVerificationMaterialDigest
	if fingerprint, fingerprintErr := openSSHPublicKeyFingerprint(publicKey); fingerprintErr != nil || fingerprint != genesisSignerFingerprint {
		keyValid = false
	}
	if !keyValid {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_commit_signature", "fallback commits require the pinned genesis signer")
		return
	}
	firstAuthorized := make(map[string]bool, len(wave1DirectMainGrantSequences["grant.authorizedPaths"]))
	for _, path := range wave1DirectMainGrantSequences["grant.authorizedPaths"] {
		firstAuthorized[path] = true
	}
	fallbackAuthorized := make(map[string]bool, len(wave1PRFallbackSequences["addendum.authorizedPaths"]))
	for _, path := range wave1PRFallbackSequences["addendum.authorizedPaths"] {
		fallbackAuthorized[path] = true
	}
	correctionAuthorized := make(map[string]bool, len(wave1MainCIFixSequences["addendum.authorizedPaths"]))
	for _, path := range wave1MainCIFixSequences["addendum.authorizedPaths"] {
		correctionAuthorized[path] = true
	}
	fixtureAuthorized := make(map[string]bool, len(wave1CIFixtureFixSequences["addendum.authorizedPaths"]))
	for _, path := range wave1CIFixtureFixSequences["addendum.authorizedPaths"] {
		fixtureAuthorized[path] = true
	}
	if fixtureFixActive && len(commits) > 4 {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_commit_count", "fixture stabilization permits exactly one successor commit")
		return
	}
	if !fixtureFixActive && mainCIFixActive && len(commits) > 3 {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_commit_count", "main-CI correction permits exactly one successor commit")
		return
	}
	previous := wave1PublishedMain
	for index, commit := range commits {
		if len(commit.parents) != 1 || commit.parents[0] != previous {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_topology", "fallback feature history must be a contiguous one-parent chain")
			return
		}
		object, err := planningGrantGitOutput(root, "cat-file", "commit", commit.id)
		if err != nil || verifyPlanningGrantCommit(object, publicKey) != nil {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_commit_signature", "every fallback feature commit must carry the pinned SSH signature")
			return
		}
		paths, err := planningGrantGitOutput(root, "diff-tree", "--no-commit-id", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "-r", commit.id+"^", commit.id)
		if err != nil {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_paths", "fallback commit paths cannot be enumerated")
			return
		}
		normalized, err := normalizedPlanningGrantGitPaths(paths)
		authorized := fallbackAuthorized
		if index == 0 {
			authorized = firstAuthorized
			if commit.id != wave1PRFallbackFirstCommit {
				addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_first_commit", "the rejected signed transition commit must remain the first exact fallback commit")
				return
			}
			tree, treeErr := planningGrantGitOutput(root, "rev-parse", "--verify", commit.id+"^{tree}")
			if treeErr != nil || strings.TrimSpace(string(tree)) != wave1PRFallbackFirstTree {
				addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_first_commit", "the rejected signed transition tree must remain unchanged")
				return
			}
		} else if index == 1 && mainCIFixActive {
			if commit.id != wave1TransitionReviewedHead {
				addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_main_ci_reviewed_head", "the reviewed changes-requested head must remain the exact second feature commit")
				return
			}
		} else if index == 2 && fixtureFixActive {
			authorized = correctionAuthorized
			if commit.id != wave1CIFixtureReviewedHead {
				addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_fixture_reviewed_head", "the failed-CI head must remain the exact third feature commit")
				return
			}
		} else if index >= 3 && fixtureFixActive {
			authorized = fixtureAuthorized
		} else if index >= 2 && mainCIFixActive {
			authorized = correctionAuthorized
		}
		if err != nil || !planningGrantPathsAllowed(normalized, authorized) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_scope", "a fallback commit includes a path outside its signed authorization phase")
			return
		}
		previous = commit.id
	}
	tracked, err := planningGrantGitOutput(root, "diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_paths", "current index and worktree paths cannot be enumerated")
		return
	}
	untracked, err := planningGrantGitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_paths", "untracked fallback paths cannot be enumerated")
		return
	}
	paths, err := normalizedPlanningGrantGitPaths(tracked, untracked)
	liveAuthorized := fallbackAuthorized
	if fixtureFixActive {
		liveAuthorized = fixtureAuthorized
	} else if mainCIFixActive {
		liveAuthorized = correctionAuthorized
	}
	if err != nil || !planningGrantPathsAllowed(paths, liveAuthorized) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_scope", "current changes include a path outside the signed PR fallback")
	}
}

func checkWave1PRFallbackCheckout(root, head string, findings *[]Finding) (planningGrantCheckout, bool) {
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		if branchErr != nil || branch != wave1PRFallbackBranch {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_branch", "local fallback work must use the exact signed branch")
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{kind: planningGrantLocalBranch, expectedHead: head}, true
	}
	if branchErr == nil && branch != "main" || branchErr != nil && branch != "" {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_branch", "GitHub fallback branch state is ambiguous")
		return planningGrantCheckout{}, false
	}
	if os.Getenv("CI") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "GitHub fallback CI lacks canonical immutable runner facts")
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "GitHub fallback run ID is invalid")
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "GitHub fallback attempt is invalid")
		return planningGrantCheckout{}, false
	}
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 {
		addFinding(findings, planningGrantWorkflowPath, "public.pr_fallback_workflow", "fallback workflow bytes do not match the pinned digest")
		return planningGrantCheckout{}, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "fallback event payload is not canonical")
		return planningGrantCheckout{}, false
	}
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) || os.Getenv("GITHUB_HEAD_REF") != wave1PRFallbackBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil || event.PullRequest.Head.Ref != wave1PRFallbackBranch ||
			event.PullRequest.Base.Ref != "main" || event.PullRequest.Base.SHA != wave1PublishedMain ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "pull-request event does not bind the exact fallback base and branch")
			return planningGrantCheckout{}, false
		}
		workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
		if workflowRef != workflowPrefix+ref && workflowRef != workflowPrefix+"refs/heads/main" {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_runner", "pull-request workflow ref is not canonical")
			return planningGrantCheckout{}, false
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 2 || parents[0] != wave1PublishedMain || parents[1] != event.PullRequest.Head.SHA {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_topology", "pull-request checkout must be the exact ordered two-parent merge")
			return planningGrantCheckout{}, false
		}
		mergeTree, mergeErr := planningGrantGitOutput(root, "rev-parse", "--verify", head+"^{tree}")
		headTree, headErr := planningGrantGitOutput(root, "rev-parse", "--verify", event.PullRequest.Head.SHA+"^{tree}")
		if mergeErr != nil || headErr != nil || strings.TrimSpace(string(mergeTree)) != strings.TrimSpace(string(headTree)) {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tree", "pull-request merge tree must equal the immutable fallback head tree")
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{kind: planningGrantPullRequestMerge, expectedHead: head, firstParent: parents[0], secondParent: parents[1]}, true
	case "push":
		if branch != "" && branch != "main" || os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" || os.Getenv("GITHUB_WORKFLOW_REF") != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.Before != wave1PublishedMain || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
			addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "protected-main event does not bind the exact fallback squash")
			return planningGrantCheckout{}, false
		}
		parents, ok := checkWave1MainSquashTopology(root, head, findings)
		if !ok {
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{kind: planningGrantMainSquash, expectedHead: head, firstParent: parents[0]}, true
	default:
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_event", "only pull-request and protected-main push events are admitted")
		return planningGrantCheckout{}, false
	}
}

func checkWave1MainSquashTopology(root, head string, findings *[]Finding) ([]string, bool) {
	parents, err := planningGrantCommitParents(root, head)
	if err != nil || len(parents) != 1 || parents[0] != wave1PublishedMain {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_topology", "protected main must contain one exact squash commit over the signed base")
		return nil, false
	}
	return parents, true
}

func checkWave1PriorTransitionTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1TransitionTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(tagObject)) != wave1TransitionTagObject {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag object must remain unchanged")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", wave1TransitionTagObject)
	if err != nil {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag cannot be read")
		return false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_v1_tag", "v1 transition tag requires the pinned genesis key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, wave1TransitionTag, wave1TransitionTagMessage)
	if err != nil || target != wave1TransitionReviewedHead {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1TransitionReviewedTree {
		addFinding(findings, wave1MainCIFixPath, "public.pr_fallback_v1_tag", "the signed v1 transition tag tree must remain unchanged")
		return false
	}
	return true
}

func checkWave1PriorV2TransitionTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1SuccessorTransitionTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(tagObject)) != wave1SuccessorTagObject {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag object must remain unchanged")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", wave1SuccessorTagObject)
	if err != nil {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag cannot be read")
		return false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_v2_tag", "v2 transition tag requires the pinned genesis key")
		return false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, wave1SuccessorTransitionTag, wave1SuccessorTagMessage)
	if err != nil || target != wave1CIFixtureReviewedHead {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1CIFixtureReviewedTree {
		addFinding(findings, wave1CIFixtureFixPath, "public.pr_fallback_v2_tag", "the signed v2 transition tag tree must remain unchanged")
		return false
	}
	return true
}

func checkWave1TransitionTag(root, tagName, tagMessage, expectedTarget, expectedTreeCommit string, requireDistinct bool, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + tagName
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "canonical CI requires the signed transition tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(tagObject)))
	if err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag cannot be read")
		return "", false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.pr_fallback_tag", "transition tag requires the pinned genesis key")
		return "", false
	}
	target, err := verifyPinnedPlanningGrantTag(object, publicKey, tagName, tagMessage)
	if err != nil || expectedTarget != "" && target != expectedTarget {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag must target the current reviewed feature head")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1TransitionReviewedHead, target); err != nil {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag target must preserve the reviewed fallback history")
		return "", false
	}
	if requireDistinct && target == expectedTreeCommit {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "squash-main commit must remain distinct from the reviewed feature head")
		return "", false
	}
	targetTree, targetErr := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	expectedTree, expectedErr := planningGrantGitOutput(root, "rev-parse", "--verify", expectedTreeCommit+"^{tree}")
	if targetErr != nil || expectedErr != nil || strings.TrimSpace(string(targetTree)) != strings.TrimSpace(string(expectedTree)) {
		addFinding(findings, wave1PRFallbackPath, "public.pr_fallback_tag", "signed transition tag tree must equal the reviewed fallback tree")
		return "", false
	}
	return target, true
}

func checkWave1V3PublicationTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1PublicationTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(tagObject)) != "b53728e3a57e6dc0d57151aa7f0bed8e44aaaa2f" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tag object must remain unchanged")
		return false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(tagObject)))
	if err != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tag cannot be read")
		return false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.direct_main_transition_v3_tag", "the signed v3 publication tag requires the pinned genesis key")
		return false
	}
	target, err := verifyPlanningGrantTag(object, publicKey)
	if err != nil || target != "7e6b765c284788442553d40792db0afb128c4872" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1PublishedTree {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_v3_tag", "the signed v3 publication tree must remain unchanged")
		return false
	}
	return true
}

func checkWave1DirectMainCheckout(root, head string, findings *[]Finding) bool {
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		if branchErr != nil || branch != "main" {
			addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_branch", "local transition work must use the exact main branch")
			return false
		}
		return true
	}
	if branchErr == nil && branch != "main" || branchErr != nil && branch != "" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_branch", "protected-main CI branch state is ambiguous")
		return false
	}
	if os.Getenv("CI") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository || os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob || os.Getenv("GITHUB_SHA") != head ||
		os.Getenv("GITHUB_EVENT_NAME") != "push" || os.Getenv("GITHUB_REF") != "refs/heads/main" ||
		os.Getenv("GITHUB_REF_PROTECTED") != "true" || os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main CI lacks canonical immutable runner facts")
		return false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main CI run ID is invalid")
		return false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main CI attempt is invalid")
		return false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	if workflowRef != planningGrantRepository+"/"+planningGrantWorkflowPath+"@refs/heads/main" {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_runner", "protected-main workflow ref is not canonical")
		return false
	}
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 {
		addFinding(findings, planningGrantWorkflowPath, "public.direct_main_transition_workflow", "protected-main workflow bytes do not match the pinned digest")
		return false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository || event.Ref != "refs/heads/main" || event.After != head ||
		event.HeadCommit == nil || event.HeadCommit.ID != head || event.PullRequest != nil {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_event", "protected-main event does not bind the exact pushed commit")
		return false
	}
	parents, err := planningGrantCommitParents(root, head)
	if err != nil || len(parents) != 1 || event.Before != parents[0] {
		addFinding(findings, wave1DirectMainGrantPath, "public.direct_main_transition_event", "protected-main event before SHA must equal the pushed commit's sole parent")
		return false
	}
	return true
}

func planningGrantCommitRangeFrom(root, base, end string) ([]planningGrantCommit, error) {
	output, err := planningGrantGitOutput(root, "rev-list", "--no-abbrev-commit", "--reverse", "--topo-order", "--parents", base+".."+end)
	if err != nil {
		return nil, err
	}
	var commits []planningGrantCommit
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !sha1Pattern.MatchString(fields[0]) || !sha1Pattern.MatchString(fields[1]) {
			return nil, fmt.Errorf("post-publication history is not a linear commit chain")
		}
		commits = append(commits, planningGrantCommit{id: fields[0], parents: []string{fields[1]}})
	}
	return commits, nil
}

func planningGrantPathsAllowed(paths []string, authorized map[string]bool) bool {
	for _, path := range paths {
		if !authorized[path] {
			return false
		}
	}
	return true
}

func checkWave1PriorPublicationTag(root string, findings *[]Finding) bool {
	ref := "refs/tags/" + wave1PriorPublicationTag
	object, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(object)) != wave1PriorPublicationTagObject {
		addFinding(findings, wave1AddendumPath, "public.prior_publication_tag", "the preserved failed v1 publication tag must resolve to its exact immutable object")
		return false
	}
	target, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{}")
	if err != nil || strings.TrimSpace(string(target)) != wave1AddendumBase {
		addFinding(findings, wave1AddendumPath, "public.prior_publication_tag", "the preserved failed v1 publication tag target must remain unchanged")
		return false
	}
	tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{}^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != wave1AddendumBaseTree {
		addFinding(findings, wave1AddendumPath, "public.prior_publication_tag", "the preserved failed v1 publication tag tree must remain unchanged")
		return false
	}
	v2Ref := "refs/tags/" + wave1V2PublicationTag
	v2Object, err := planningGrantGitOutput(root, "rev-parse", "--verify", v2Ref+"^{tag}")
	if err != nil || strings.TrimSpace(string(v2Object)) != wave1V2PublicationTagObject {
		addFinding(findings, wave1V3AddendumPath, "public.prior_v2_publication_tag", "the preserved failed v2 publication tag must resolve to its exact immutable object")
		return false
	}
	v2Target, err := planningGrantGitOutput(root, "rev-parse", "--verify", v2Ref+"^{}")
	if err != nil || strings.TrimSpace(string(v2Target)) != wave1V3AddendumBase {
		addFinding(findings, wave1V3AddendumPath, "public.prior_v2_publication_tag", "the preserved failed v2 publication tag target must remain unchanged")
		return false
	}
	v2Tree, err := planningGrantGitOutput(root, "rev-parse", "--verify", v2Ref+"^{}^{tree}")
	if err != nil || strings.TrimSpace(string(v2Tree)) != wave1V3AddendumBaseTree {
		addFinding(findings, wave1V3AddendumPath, "public.prior_v2_publication_tag", "the preserved failed v2 publication tag tree must remain unchanged")
		return false
	}
	return true
}

// Contract publication has exactly three admissible checkout states:
//
//   - A local pre-merge checkout must be the exact symbolic branch named by the
//     signed grant. It must remain linear, so an unsigned local merge cannot
//     exploit the deliberate GitHub merge-signature exception.
//   - Pull-request CI must be detached at GitHub's two-parent synthetic merge.
//     Canonical runner facts and the bounded event payload bind HEAD to the
//     immutable base/head SHAs and to the signed feature branch.
//   - Main-push CI may be detached or symbolic main. The one-parent squash
//     commit must have the event's exact protected base and the same tree as a
//     pinned-signer annotated tag. That tag keeps the reviewed signed feature
//     history reachable after GitHub rewrites the public commit identity.
//
// There is no generic detached, arbitrary-branch, or local-main fallback.
func planningGrantCheckoutState(root, head string, findings *[]Finding) (planningGrantCheckout, bool) {
	branchOutput, branchErr := planningGrantGitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(string(branchOutput))
	if branchErr == nil && branch == wave1PlanningGrantBranch {
		return planningGrantCheckout{kind: planningGrantLocalBranch, expectedHead: head}, true
	}
	if branchErr == nil && branch != "main" {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_branch", "contract publication must run on the exact signed working branch")
		return planningGrantCheckout{}, false
	}
	if branchErr != nil && branch != "" {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_branch", "symbolic branch state is ambiguous")
		return planningGrantCheckout{}, false
	}

	checkout, ok := planningGrantGitHubCheckout(root, head, branch)
	if !ok {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_branch", "detached or main checkout lacks canonical immutable workflow facts")
		return planningGrantCheckout{}, false
	}
	return checkout, true
}

func planningGrantGitHubCheckout(root, head, branch string) (planningGrantCheckout, bool) {
	if os.Getenv("CI") != "true" || os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" ||
		os.Getenv("GITHUB_REPOSITORY") != planningGrantRepository ||
		os.Getenv("GITHUB_WORKFLOW") != planningGrantWorkflow ||
		os.Getenv("GITHUB_JOB") != planningGrantWorkflowJob ||
		os.Getenv("GITHUB_SHA") != head ||
		!samePlanningGrantRepositoryRoot(root, os.Getenv("GITHUB_WORKSPACE")) {
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ID")); !ok {
		return planningGrantCheckout{}, false
	}
	if _, ok := parsePositiveInt(os.Getenv("GITHUB_RUN_ATTEMPT")); !ok {
		return planningGrantCheckout{}, false
	}
	workflowRef := os.Getenv("GITHUB_WORKFLOW_REF")
	workflowPrefix := planningGrantRepository + "/" + planningGrantWorkflowPath + "@"
	if !strings.HasPrefix(workflowRef, workflowPrefix) {
		return planningGrantCheckout{}, false
	}
	workflow, err := readRepoFile(root, planningGrantWorkflowPath)
	if err != nil || fileSHA256(workflow) != canonicalFoundationWorkflowSHA256 {
		return planningGrantCheckout{}, false
	}
	event, ok := readPlanningGrantGitHubEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if !ok || event.Repository.FullName != planningGrantRepository {
		return planningGrantCheckout{}, false
	}

	switch os.Getenv("GITHUB_EVENT_NAME") {
	case "pull_request":
		ref := os.Getenv("GITHUB_REF")
		if branch != "" || !validPlanningGrantPullRequestRef(ref) ||
			os.Getenv("GITHUB_HEAD_REF") != wave1PlanningGrantBranch ||
			os.Getenv("GITHUB_BASE_REF") != "main" || event.PullRequest == nil ||
			event.PullRequest.Head.Ref != wave1PlanningGrantBranch ||
			event.PullRequest.Base.Ref != "main" ||
			!validAdvisoryPullRequestMergeSHA(event.PullRequest.MergeCommitSHA) ||
			!sha1Pattern.MatchString(event.PullRequest.Head.SHA) ||
			!sha1Pattern.MatchString(event.PullRequest.Base.SHA) {
			return planningGrantCheckout{}, false
		}
		workflowSuffix := strings.TrimPrefix(workflowRef, workflowPrefix)
		if workflowSuffix != ref && workflowSuffix != "refs/heads/main" {
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{
			kind:         planningGrantPullRequestMerge,
			expectedHead: head,
			firstParent:  event.PullRequest.Base.SHA,
			secondParent: event.PullRequest.Head.SHA,
		}, true
	case "push":
		if branch != "" && branch != "main" {
			return planningGrantCheckout{}, false
		}
		if os.Getenv("GITHUB_REF") != "refs/heads/main" || os.Getenv("GITHUB_REF_PROTECTED") != "true" ||
			os.Getenv("GITHUB_HEAD_REF") != "" || os.Getenv("GITHUB_BASE_REF") != "" ||
			workflowRef != workflowPrefix+"refs/heads/main" ||
			event.Ref != "refs/heads/main" || event.After != head || event.HeadCommit == nil || event.HeadCommit.ID != head ||
			event.Before != wave1PlanningGrantBase {
			return planningGrantCheckout{}, false
		}
		return planningGrantCheckout{
			kind:         planningGrantMainSquash,
			expectedHead: head,
			firstParent:  event.Before,
		}, true
	default:
		return planningGrantCheckout{}, false
	}
}

func readPlanningGrantGitHubEvent(path string) (planningGrantGitHubEvent, bool) {
	var event planningGrantGitHubEvent
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
		return event, false
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &event) != nil {
		return event, false
	}
	return event, true
}

func validPlanningGrantPullRequestRef(ref string) bool {
	parts := strings.Split(strings.TrimPrefix(ref, "refs/pull/"), "/")
	if len(parts) != 2 || parts[1] != "merge" {
		return false
	}
	_, ok := parsePositiveInt(parts[0])
	return ok
}

func validAdvisoryPullRequestMergeSHA(value string) bool {
	return value == "" || sha1Pattern.MatchString(value)
}

func planningGrantPublicationTagTarget(root string, checkout planningGrantCheckout, head string, findings *[]Finding) (string, bool) {
	ref := "refs/tags/" + wave1PublicationTag
	tagObject, err := planningGrantGitOutput(root, "rev-parse", "--verify", ref+"^{tag}")
	if err != nil || !sha1Pattern.MatchString(strings.TrimSpace(string(tagObject))) {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_missing", "canonical CI requires the signed immutable Wave-1 publication tag")
		return "", false
	}
	object, err := planningGrantGitOutput(root, "cat-file", "tag", strings.TrimSpace(string(tagObject)))
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_object", "publication tag object cannot be read")
		return "", false
	}
	publicKey, err := readRepoFile(root, wave1PlanningGrantKey)
	if err != nil || fileSHA256(publicKey) != genesisVerificationMaterialDigest {
		addFinding(findings, wave1PlanningGrantKey, "public.publication_tag_key", "publication tag requires the pinned genesis signer")
		return "", false
	}
	target, err := verifyPlanningGrantTag(object, publicKey)
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_signature", "publication tag is not an exact pinned-signer tree attestation")
		return "", false
	}
	if target == wave1PlanningGrantBase {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_target", "publication tag must attest nonempty reviewed feature history")
		return "", false
	}
	if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", wave1PlanningGrantBase, target); err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_target", "publication tag target must descend from the exact signed base")
		return "", false
	}
	targetTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", target+"^{tree}")
	if err != nil {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_tree", "publication tag target tree cannot be resolved")
		return "", false
	}
	expectedCommit := head
	if checkout.kind == planningGrantPullRequestMerge {
		expectedCommit = checkout.secondParent
		if _, err := planningGrantGitOutput(root, "merge-base", "--is-ancestor", target, checkout.secondParent); err != nil {
			addFinding(findings, wave1DispositionPath, "public.publication_tag_target", "publication tag target must be an ancestor of the immutable pull-request head")
			return "", false
		}
	}
	expectedTree, err := planningGrantGitOutput(root, "rev-parse", "--verify", expectedCommit+"^{tree}")
	if err != nil || strings.TrimSpace(string(targetTree)) != strings.TrimSpace(string(expectedTree)) {
		addFinding(findings, wave1DispositionPath, "public.publication_tag_tree", "signed publication tag tree must equal the reviewed publication tree")
		return "", false
	}
	return target, true
}

func verifyPlanningGrantTag(object, publicKey []byte) (string, error) {
	return verifyPinnedPlanningGrantTag(object, publicKey, wave1PublicationTag, wave1PublicationTagMessage)
}

func verifyPinnedPlanningGrantTag(object, publicKey []byte, expectedTag, expectedMessage string) (string, error) {
	return verifyPinnedPlanningGrantTagForIdentity(object, publicKey, expectedTag, expectedMessage, "release-manager@example.com")
}

func verifyPinnedPlanningGrantTagForIdentity(object, publicKey []byte, expectedTag, expectedMessage, expectedEmail string) (string, error) {
	const armor = "-----BEGIN SSH SIGNATURE-----"
	signatureIndex := bytes.Index(object, []byte(armor))
	if signatureIndex < 0 || bytes.Count(object, []byte(armor)) != 1 {
		return "", fmt.Errorf("tag must contain exactly one SSH signature")
	}
	signed := object[:signatureIndex]
	signature := object[signatureIndex:]
	if err := verifySSHSig(signed, signature, publicKey, planningGrantCommitNS); err != nil {
		return "", err
	}
	headerEnd := bytes.Index(signed, []byte("\n\n"))
	if headerEnd < 0 {
		return "", fmt.Errorf("tag object has no header boundary")
	}
	var target string
	counts := map[string]int{}
	for _, line := range strings.Split(string(signed[:headerEnd]), "\n") {
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			return "", fmt.Errorf("invalid tag header")
		}
		counts[fields[0]]++
		switch fields[0] {
		case "object":
			target = fields[1]
		case "type":
			if fields[1] != "commit" {
				return "", fmt.Errorf("tag target must be a commit")
			}
		case "tag":
			if fields[1] != expectedTag {
				return "", fmt.Errorf("tag name does not match the signed recovery disposition")
			}
		case "tagger":
			if !strings.Contains(fields[1], " <"+expectedEmail+"> ") {
				return "", fmt.Errorf("tagger must use the required synthetic public identity")
			}
		default:
			return "", fmt.Errorf("unexpected tag header")
		}
	}
	for _, field := range []string{"object", "type", "tag", "tagger"} {
		if counts[field] != 1 {
			return "", fmt.Errorf("tag header cardinality is invalid")
		}
	}
	if !sha1Pattern.MatchString(target) {
		return "", fmt.Errorf("tag target is not a canonical commit ID")
	}
	if string(signed[headerEnd+2:]) != expectedMessage+"\n" {
		return "", fmt.Errorf("tag message does not match the signed recovery disposition")
	}
	return target, nil
}

func planningGrantCommitRange(root, end string) ([]planningGrantCommit, error) {
	output, err := planningGrantGitOutput(root, "rev-list", "--no-abbrev-commit", "--reverse", "--topo-order", "--parents", wave1PlanningGrantBase+".."+end)
	if err != nil {
		return nil, err
	}
	var commits []planningGrantCommit
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !sha1Pattern.MatchString(fields[0]) {
			return nil, fmt.Errorf("invalid commit ancestry record")
		}
		commit := planningGrantCommit{id: fields[0]}
		for _, parent := range fields[1:] {
			if !sha1Pattern.MatchString(parent) {
				return nil, fmt.Errorf("invalid commit parent")
			}
			commit.parents = append(commit.parents, parent)
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

func planningGrantCommitParents(root, commit string) ([]string, error) {
	output, err := planningGrantGitOutput(root, "rev-list", "--no-abbrev-commit", "--parents", "-n", "1", commit)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 1 || fields[0] != commit {
		return nil, fmt.Errorf("invalid commit parent record")
	}
	parents := make([]string, 0, len(fields)-1)
	for _, parent := range fields[1:] {
		if !sha1Pattern.MatchString(parent) {
			return nil, fmt.Errorf("invalid commit parent")
		}
		parents = append(parents, parent)
	}
	return parents, nil
}

func checkPlanningGrantCommitTopology(root string, checkout planningGrantCheckout, head string, commits []planningGrantCommit, findings *[]Finding) bool {
	if checkout.kind == planningGrantLocalBranch {
		for _, commit := range commits {
			if len(commit.parents) != 1 {
				addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "the pre-merge signed branch must have linear commit history")
				return false
			}
		}
		return true
	}
	if checkout.kind == planningGrantMainSquash {
		if len(commits) == 0 || commits[len(commits)-1].id != checkout.tagTarget {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "signed publication tag must terminate the preserved feature history")
			return false
		}
		for _, commit := range commits {
			if len(commit.parents) != 1 {
				addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "tag-preserved feature history must be linear")
				return false
			}
		}
		parents, err := planningGrantCommitParents(root, head)
		if err != nil || len(parents) != 1 || parents[0] != checkout.firstParent {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "protected main must contain one exact squash commit over the recorded base")
			return false
		}
		return true
	}
	if len(commits) == 0 || commits[len(commits)-1].id != head {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "canonical CI must end at its declared merge commit")
		return false
	}
	for _, commit := range commits[:len(commits)-1] {
		if len(commit.parents) != 1 {
			addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "only the canonical terminal CI merge may be unsigned")
			return false
		}
	}
	merge := commits[len(commits)-1]
	if len(merge.parents) != 2 || merge.parents[0] != checkout.firstParent {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "canonical CI requires the exact two-parent merge recorded by the runner event")
		return false
	}
	if checkout.kind == planningGrantPullRequestMerge && merge.parents[1] != checkout.secondParent {
		addFinding(findings, wave1PlanningGrantPath, "public.planning_grant_commit_topology", "pull-request merge does not contain the immutable signed feature head")
		return false
	}
	return true
}

func verifyPlanningGrantCommit(object, publicKey []byte) error {
	// Git SSH-signs the commit payload under the protocol-defined "git"
	// namespace. The detached grant document is independently verified under
	// mars3-planning-grant; accepting either namespace for both artifacts would
	// make a valid signature replayable across authority surfaces.
	headerEnd := bytes.Index(object, []byte("\n\n"))
	if headerEnd < 0 {
		return fmt.Errorf("commit object has no header boundary")
	}
	headers := bytes.Split(object[:headerEnd], []byte("\n"))
	var signed bytes.Buffer
	var signature bytes.Buffer
	signatureCount := 0
	for index := 0; index < len(headers); index++ {
		line := headers[index]
		if !bytes.HasPrefix(line, []byte("gpgsig ")) {
			signed.Write(line)
			signed.WriteByte('\n')
			continue
		}
		signatureCount++
		signature.Write(bytes.TrimPrefix(line, []byte("gpgsig ")))
		signature.WriteByte('\n')
		for index+1 < len(headers) && bytes.HasPrefix(headers[index+1], []byte(" ")) {
			index++
			signature.Write(headers[index][1:])
			signature.WriteByte('\n')
		}
	}
	if signatureCount != 1 {
		return fmt.Errorf("commit must have exactly one signature header")
	}
	signed.WriteByte('\n')
	signed.Write(object[headerEnd+2:])
	return verifySSHSig(signed.Bytes(), signature.Bytes(), publicKey, planningGrantCommitNS)
}

func planningGrantGitOutput(root string, arguments ...string) ([]byte, error) {
	prefix := []string{"-c", "core.fsmonitor=false", "-c", "diff.external=", "-C", root}
	command := exec.Command("/usr/bin/git", append(prefix, arguments...)...)
	command.Env = planningGrantGitEnvironment()
	return command.Output()
}

func planningGrantGitEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+5)
	for _, entry := range os.Environ() {
		key := entry
		if separator := strings.IndexByte(entry, '='); separator >= 0 {
			key = entry[:separator]
		}
		if strings.HasPrefix(key, "GIT_") {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment,
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"LC_ALL=C",
	)
}

func samePlanningGrantRepositoryRoot(root, topLevel string) bool {
	if topLevel == "" {
		return false
	}
	rootInfo, rootErr := os.Stat(root)
	topInfo, topErr := os.Stat(topLevel)
	if rootErr != nil || topErr != nil {
		return false
	}
	return os.SameFile(rootInfo, topInfo)
}

func normalizedPlanningGrantGitPaths(outputs ...[]byte) ([]string, error) {
	unique := make(map[string]bool)
	for _, output := range outputs {
		if len(output) == 0 {
			continue
		}
		if output[len(output)-1] != 0 {
			return nil, fmt.Errorf("Git path output is not NUL terminated")
		}
		for _, encoded := range bytes.Split(output[:len(output)-1], []byte{0}) {
			if len(encoded) == 0 || !utf8.Valid(encoded) {
				return nil, fmt.Errorf("Git returned an empty or non-UTF-8 path")
			}
			path := string(encoded)
			if !safeRelativePath(path) || filepath.ToSlash(filepath.Clean(path)) != path {
				return nil, fmt.Errorf("Git returned a non-canonical path")
			}
			unique[path] = true
		}
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func parseStrictPlanningGrant(data []byte) strictPlanningGrant {
	return parseStrictGrant(data, wave1PlanningGrantScalars, wave1PlanningGrantSequences, []string{"grant", "verification", "integrity"})
}

func parseStrictGrant(data []byte, expectedScalars []grantScalarExpectation, expectedSequences map[string][]string, expectedSections []string) strictPlanningGrant {
	document := strictPlanningGrant{
		scalars:         make(map[string][]string),
		sequences:       make(map[string][]string),
		sections:        make(map[string]int),
		sequenceHeaders: make(map[string]int),
	}
	if !utf8.Valid(data) {
		document.structuralErrors = append(document.structuralErrors, "grant must be valid UTF-8")
		return document
	}

	knownScalars := make(map[string]bool, len(expectedScalars))
	for _, expected := range expectedScalars {
		knownScalars[expected.path] = true
	}
	knownSequences := make(map[string]bool, len(expectedSequences))
	for path := range expectedSequences {
		knownSequences[path] = true
	}
	knownSections := make(map[string]bool, len(expectedSections))
	for _, section := range expectedSections {
		knownSections[section] = true
	}

	section := ""
	sequence := ""
	for index, raw := range strings.Split(string(data), "\n") {
		lineNumber := index + 1
		raw = strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if strings.Contains(raw, "\t") {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses a tab", lineNumber))
			continue
		}
		if strings.HasPrefix(trimmed, "#") || trimmed == "---" || trimmed == "..." || strings.HasPrefix(trimmed, "%") {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses unsupported YAML syntax", lineNumber))
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if strings.HasPrefix(trimmed, "-") {
			if indent != 4 || sequence == "" || !strings.HasPrefix(trimmed, "- ") {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d is not a scalar item of a declared sequence", lineNumber))
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if item == "" || unsafeGrantYAMLValue(item) {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses an empty or indirect sequence value", lineNumber))
				continue
			}
			document.sequences[sequence] = append(document.sequences[sequence], item)
			continue
		}

		sequence = ""
		colon := strings.Index(trimmed, ":")
		if colon < 1 {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d is not a plain mapping entry", lineNumber))
			continue
		}
		key := strings.TrimSpace(trimmed[:colon])
		value := strings.TrimSpace(trimmed[colon+1:])
		if key == "" || unsafeGrantYAMLKey(key) || unsafeGrantYAMLValue(value) {
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d uses YAML indirection or an unsafe mapping", lineNumber))
			continue
		}

		switch indent {
		case 0:
			section = ""
			if knownSections[key] {
				if value != "" {
					document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d must open a mapping", lineNumber))
					continue
				}
				document.sections[key]++
				section = key
				continue
			}
			if !knownScalars[key] {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d declares an unknown root field", lineNumber))
				continue
			}
			if value == "" {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d requires a scalar value", lineNumber))
				continue
			}
			document.scalars[key] = append(document.scalars[key], value)
		case 2:
			if section == "" {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d has no declared parent mapping", lineNumber))
				continue
			}
			path := section + "." + key
			if knownSequences[path] {
				if value != "" {
					document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d must open a block scalar sequence", lineNumber))
					continue
				}
				document.sequenceHeaders[path]++
				sequence = path
				continue
			}
			if !knownScalars[path] {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d declares an unknown contract field", lineNumber))
				continue
			}
			if value == "" {
				document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d requires a scalar value", lineNumber))
				continue
			}
			document.scalars[path] = append(document.scalars[path], value)
		default:
			document.structuralErrors = append(document.structuralErrors, fmt.Sprintf("line %d has unsupported nesting", lineNumber))
		}
	}
	return document
}

func unsafeGrantYAMLKey(value string) bool {
	return strings.ContainsAny(value, "&*!{}[]|><#") || strings.Contains(value, "<<")
}

func unsafeGrantYAMLValue(value string) bool {
	if value == "" {
		return false
	}
	if strings.Contains(value, "<<:") || strings.ContainsAny(value, "{}[]") {
		return true
	}
	for _, prefix := range []string{"&", "*", "!", "|", ">", "'", "\""} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	for _, token := range []string{" &", " *", " !"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func equalStringSequence(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}
