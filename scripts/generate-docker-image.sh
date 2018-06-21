#!/bin/bash

main(){
	echo "Creating Eirini docker image..."
	build_opi
	create_eirinifs
	create_docker_images
	echo "Eirini docker image created"
}

build_opi(){
	GOOS=linux CGO_ENABLED=0 go build -a -o image/opi/opi code.cloudfoundry.org/eirini/cmd/opi
	verify_exit_code $? "Failed to build eirini"
  cp image/opi/opi image/registry/opi
}

create_eirinifs(){
	./launcher/bin/build-eirinifs.sh && \
	cp launcher/image/eirinifs.tar ./image/registry/

	verify_exit_code $? "Failed to create eirinifs.tar"
}

create_docker_images() {
	pushd ./image/opi
	docker build . -t eirini/opi
	verify_exit_code $? "Failed to create opi docker image"
  popd

  pushd ./image/registry
  docker build . -t eirini/registry
	verify_exit_code $? "Failed to create registry docker image"
  popd
}

verify_exit_code() {
	local exit_code=$1
	local error_msg=$2
	if [ "$exit_code" -ne 0 ]; then
		echo "$error_msg"
		exit 1
	fi
}









main
