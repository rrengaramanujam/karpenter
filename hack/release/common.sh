#!/usr/bin/env bash
set -euo pipefail

config(){
  GITHUB_ACCOUNT="aws"
  AWS_ACCOUNT_ID="071440425669"
  ECR_GALLERY_NAME="karpenter"
  RELEASE_REPO_ECR=${RELEASE_REPO_ECR:-public.ecr.aws/${ECR_GALLERY_NAME}/}
  RELEASE_REPO_GH=${RELEASE_REPO_GH:-ghcr.io/${GITHUB_ACCOUNT}/karpenter}

  PRIVATE_PULL_THROUGH_HOST="${AWS_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com"
  CURRENT_MAJOR_VERSION="0"
  RELEASE_PLATFORM="--platform=linux/amd64,linux/arm64"

  MAIN_GITHUB_ACCOUNT="aws"
  RELEASE_TYPE_STABLE="stable"
  RELEASE_TYPE_SNAPSHOT="snapshot"
}

# versionData sets all the version properties for the passed release version. It sets the values
# RELEASE_VERSION_MAJOR, RELEASE_VERSION_MINOR, and RELEASE_VERSION_PATCH to be used by other scripts
versionData(){
  local VERSION="$1"
  local VERSION="${VERSION#[vV]}"
  RELEASE_VERSION_MAJOR="${VERSION%%\.*}"
  RELEASE_VERSION_MINOR="${VERSION#*.}"
  RELEASE_VERSION_MINOR="${RELEASE_VERSION_MINOR%.*}"
  RELEASE_VERSION_PATCH="${VERSION##*.}"
  RELEASE_MINOR_VERSION="v${RELEASE_VERSION_MAJOR}.${RELEASE_VERSION_MINOR}"
}

release() {
  RELEASE_VERSION=$1
  echo "Release Type: $(releaseType "${RELEASE_VERSION}")
Release Version: ${RELEASE_VERSION}
Commit: $(git rev-parse HEAD)
Helm Chart Version $(helmChartVersion $RELEASE_VERSION)"

  authenticate
  buildImages
  cosignImages
  publishHelmChart "karpenter" "${RELEASE_VERSION}" "${RELEASE_REPO_ECR}"
  publishHelmChart "karpenter-crd" "${RELEASE_VERSION}" "${RELEASE_REPO_ECR}"
  pullPrivateReplica "$RELEASE_VERSION"
}

authenticate() {
  aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin "${RELEASE_REPO_ECR}"
}

authenticatePrivateRepo() {
  aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin ${PRIVATE_PULL_THROUGH_HOST}
}

buildImages() {
    # Set the SOURCE_DATE_EPOCH and KO_DATA_DATE_EPOCH values for reproducable builds with timestamps
    # https://ko.build/advanced/faq/
    CONTROLLER_IMG=$(GOFLAGS=${GOFLAGS} SOURCE_DATE_EPOCH=$(git log -1 --format='%ct') KO_DATA_DATE_EPOCH=$(git log -1 --format='%ct') KO_DOCKER_REPO=${RELEASE_REPO_ECR} ko publish -B -t "${RELEASE_VERSION}" "${RELEASE_PLATFORM}" ./cmd/controller)
    HELM_CHART_VERSION=$(helmChartVersion "$RELEASE_VERSION")
    IMG_REPOSITORY=$(echo "$CONTROLLER_IMG" | cut -d "@" -f 1 | cut -d ":" -f 1)
    IMG_TAG=$(echo "$CONTROLLER_IMG" | cut -d "@" -f 1 | cut -d ":" -f 2 -s)
    IMG_DIGEST=$(echo "$CONTROLLER_IMG" | cut -d "@" -f 2)
    yq e -i ".controller.image.repository = \"${IMG_REPOSITORY}\"" charts/karpenter/values.yaml
    yq e -i ".controller.image.tag = \"${IMG_TAG}\"" charts/karpenter/values.yaml
    yq e -i ".controller.image.digest = \"${IMG_DIGEST}\"" charts/karpenter/values.yaml
    yq e -i ".appVersion = \"${RELEASE_VERSION#v}\"" charts/karpenter/Chart.yaml
    yq e -i ".version = \"${HELM_CHART_VERSION#v}\"" charts/karpenter/Chart.yaml
    yq e -i ".appVersion = \"${RELEASE_VERSION#v}\"" charts/karpenter-crd/Chart.yaml
    yq e -i ".version = \"${HELM_CHART_VERSION#v}\"" charts/karpenter-crd/Chart.yaml
}

releaseType(){
  RELEASE_VERSION=$1

  if [[ "${RELEASE_VERSION}" == v* ]]; then
    echo $RELEASE_TYPE_STABLE
  else
    echo $RELEASE_TYPE_SNAPSHOT
  fi
}

helmChartVersion(){
    RELEASE_VERSION=$1
    if [[ $(releaseType "$RELEASE_VERSION") == "$RELEASE_TYPE_STABLE" ]]; then
      echo "$RELEASE_VERSION"
    fi

    if [[ $(releaseType "$RELEASE_VERSION") == "$RELEASE_TYPE_SNAPSHOT" ]]; then
      echo "v${CURRENT_MAJOR_VERSION}-${RELEASE_VERSION}"
    fi
}

buildDate(){
    # Set the SOURCE_DATE_EPOCH and KO_DATA_DATE_EPOCH values for reproducable builds with timestamps
    # https://ko.build/advanced/faq/
    DATE_FMT="+%Y-%m-%dT%H:%M:%SZ"
    SOURCE_DATE_EPOCH=$(git log -1 --format='%ct')
    echo "$(date -u -r "${SOURCE_DATE_EPOCH}" $DATE_FMT 2>/dev/null)"
}

cosignImages() {
    COSIGN_EXPERIMENTAL=1 cosign sign \
        -a GIT_HASH="$(git rev-parse HEAD)" \
        -a GIT_VERSION="${RELEASE_VERSION}" \
        -a BUILD_DATE="$(buildDate)" \
        "${CONTROLLER_IMG}"
}

pullPrivateReplica(){
  authenticatePrivateRepo
  RELEASE_IDENTIFIER=$1
  PULL_THROUGH_CACHE_PATH="${PRIVATE_PULL_THROUGH_HOST}/ecr-public/${ECR_GALLERY_NAME}/"
  HELM_CHART_VERSION=$(helmChartVersion "$RELEASE_VERSION")
  docker pull "${PULL_THROUGH_CACHE_PATH}controller:${RELEASE_IDENTIFIER}"
  helm pull "oci://${PULL_THROUGH_CACHE_PATH}karpenter" --version "${HELM_CHART_VERSION}"
  helm pull "oci://${PULL_THROUGH_CACHE_PATH}karpenter-crd" --version "${HELM_CHART_VERSION}"
}

publishHelmChart() {
    CHART_NAME=$1
    RELEASE_VERSION=$2
    RELEASE_REPO=$3
    HELM_CHART_VERSION=$(helmChartVersion "$RELEASE_VERSION")
    HELM_CHART_FILE_NAME="${CHART_NAME}-${HELM_CHART_VERSION}.tgz"

    cd charts
    helm dependency update "${CHART_NAME}"
    helm lint "${CHART_NAME}"
    helm package "${CHART_NAME}" --version "$HELM_CHART_VERSION"
    helm push "${HELM_CHART_FILE_NAME}" "oci://${RELEASE_REPO}"
    rm "${HELM_CHART_FILE_NAME}"
    cd ..
}

createNewWebsiteDirectory() {
    RELEASE_VERSION=$1
    versionData "${RELEASE_VERSION}"
    
    mkdir -p "website/content/en/${RELEASE_MINOR_VERSION}"
    cp -r website/content/en/preview/* "website/content/en/${RELEASE_MINOR_VERSION}/"
    find "website/content/en/${RELEASE_MINOR_VERSION}/" -type f | xargs perl -i -p -e "s/{{< param \"latest_release_version\" >}}/${RELEASE_VERSION}/g;"
    find website/content/en/${RELEASE_MINOR_VERSION}/*/*/*.yaml -type f | xargs perl -i -p -e "s/preview/${RELEASE_MINOR_VERSION}/g;"
    find "website/content/en/${RELEASE_MINOR_VERSION}/" -type f | xargs perl -i -p -e "s/{{< githubRelRef >}}/\/${RELEASE_VERSION}\//g;"

    rm -rf website/content/en/docs
    mkdir -p website/content/en/docs
    cp -r website/content/en/${RELEASE_MINOR_VERSION}/* website/content/en/docs/
}

removeOldWebsiteDirectories() {
  local n=5
  # Get all the directories except the last n directories sorted from earliest to latest version
  last_n_versions=$(find website/content/en/* -type d -name "*" -maxdepth 0 | grep -v "preview\|docs" | sort | tail -n "$n")
  last_n_versions+=$(echo -e "\nwebsite/content/en/preview")
  last_n_versions+=$(echo -e "\nwebsite/content/en/docs")
  all=$(find website/content/en/* -type d -name "*" -maxdepth 0)
  ## symmetric difference
  comm -3 <(sort <<< $last_n_versions) <(sort <<< $all) | tr -d '\t' | xargs -r -n 1 rm -r
}

editWebsiteConfig() {
  RELEASE_VERSION=$1
  yq -i ".params.latest_release_version = \"${RELEASE_VERSION}\"" website/config.yaml
}

# editWebsiteVersionsMenu sets relevant releases in the version dropdown menu of the website
# without increasing the size of the set.
editWebsiteVersionsMenu() {
  RELEASE_VERSION=$1
  versionData "${RELEASE_VERSION}"
  VERSIONS=("${RELEASE_MINOR_VERSION}")
  while IFS= read -r LINE; do
    SANITIZED_VERSION=$(echo "${LINE}" | sed -e 's/["-]//g' -e 's/ *//g')
    # Bail from this config.yaml update if the version is already in the existing list
    if [[ "${RELEASE_MINOR_VERSION}" == "${SANITIZED_VERSION}" ]]; then
      return
    fi
    VERSIONS+=("${SANITIZED_VERSION}")
  done < <(yq '.params.versions' website/config.yaml)
  unset VERSIONS[${#VERSIONS[@]}-2]

  yq -i '.params.versions = []' website/config.yaml

  for VERSION in "${VERSIONS[@]}"; do
    yq -i ".params.versions += \"${VERSION}\"" website/config.yaml
  done
}
