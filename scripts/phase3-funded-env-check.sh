#!/usr/bin/env bash
# Warn if environment looks like a DEV-ONLY funded path when aiming for production-shaped Phase 3.
# Exit 0 always (advisory). See research/PHASE_3_OPERATOR_CHECKLIST.md.
#
# Usage: ./scripts/phase3-funded-env-check.sh

set -uo pipefail

warned=0

if [[ "${E2E_MWEB_FUNDED_DEV_CLEAR:-}" == "1" ]] || [[ "${E2E_MWEB_FUNDED_DEV_CLEAR:-}" == "true" ]]; then
	echo "warning: E2E_MWEB_FUNDED_DEV_CLEAR is set — pendingOnions may clear without chain finalize (DEV ONLY)." >&2
	warned=1
fi

if [[ "${PHASE3_ALLOW_DEV_CLEAR:-}" != "1" ]] && [[ "${warned}" -eq 1 ]]; then
	echo "  Unset E2E_MWEB_FUNDED_DEV_CLEAR for README Phase 3 success bar (real finalize)." >&2
fi

if [[ "${E2E_MWEB_FUNDED:-}" == "1" ]] || [[ "${E2E_MWEB_FUNDED:-}" == "true" ]]; then
	if [[ -z "${MWEB_SCAN_SECRET:-}" ]] || [[ -z "${MWEB_SPEND_SECRET:-}" ]]; then
		echo "warning: E2E_MWEB_FUNDED=1 but MWEB_SCAN_SECRET or MWEB_SPEND_SECRET is empty." >&2
	fi
	if [[ -z "${COINSWAPD_FEE_MWEB:-}" ]] && [[ -z "${E2E_MWEB_DEST:-}" ]]; then
		echo "warning: set COINSWAPD_FEE_MWEB (maker fork -a) and/or E2E_MWEB_* per PHASE_3_MWEB_HANDOFF_SLICE.md." >&2
	fi
fi

exit 0
