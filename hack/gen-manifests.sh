#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")

mkdir -p "${SCRIPT_ROOT}/_output/"{manifests,tls,bin}
host=${1:-cel-shim-webhook.default.svc}
SECRET_NAME="${SECRET_NAME:-cel-shim-webhook.tls.example.com}"

GENTLS="go run ${SCRIPT_ROOT}/cmd/gentls"
echo "generating TLS keypair for ${host}"
# (cd "${SCRIPT_ROOT}/_output/tls" && ${GENTLS} -host="${host}")
CA_PEM=$(base64 -w0 < "${SCRIPT_ROOT}/_output/tls/ca.pem")
export CA_PEM

echo "---" > "${SCRIPT_ROOT}/_output/manifests/tls-secret.yaml"
echo "creating secret ${SECRET_NAME}"
(cd "${SCRIPT_ROOT}/_output/tls" && \
  kubectl create secret tls "${SECRET_NAME}" --cert=server.pem --key=server-key.pem --dry-run=client -oyaml >> "${SCRIPT_ROOT}/_output/manifests/tls-secret.yaml")

echo "copying manifests"
cp "${SCRIPT_ROOT}/manifests/"*.yaml "${SCRIPT_ROOT}/_output/manifests"

go run github.com/mikefarah/yq/v4 eval -i ".webhooks[0].clientConfig.caBundle = env(CA_PEM)" "${SCRIPT_ROOT}/_output/manifests/webhook-config.yaml"
