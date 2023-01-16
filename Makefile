VERSION:=$VERSION

COMMON_FILES:=$(shell find 'common' ! -iname '*_test.go')
VENDOR_FOLDER:=./vendor
PACKAGING_FOLDER:=target/package
TARGET_FOLDER:=target
MOD_FILES:= \
	go.sum \
	go.mod

#Audit Files
AUDIT_ENTRY_PT:= \
	./cloud-functions/audit/audit_pusher.go

AUDIT_FILES:= \
	./cloud-functions/retailers/models/retailer.go \
	./cloud-functions/sites/models/site.go


# Retailer Package Information

GET_RETAILER_ENTRY_PT:= \
	./cloud-functions/retailers/get_retailer.go

GET_RETAILER_FILES:= \
	./cloud-functions/retailers/models/retailer.go

POST_RETAILER_ENTRY_PT:= \
	./cloud-functions/retailers/post_retailer.go \

POST_RETAILER_FILES:= \
	./cloud-functions/retailers/models/retailer.go

GET_RETAILERS_ENTRY_PT:= \
	./cloud-functions/retailers/get_retailers.go

GET_RETAILERS_FILES:= \
	./cloud-functions/retailers/models/retailer.go

GET_RETAILER_AUDIT_ENTRY_PT:= \
	./cloud-functions/retailers/get_retailer_audit.go

GET_RETAILER_AUDIT_FILES:= \
	./cloud-functions/retailers/models/retailer.go

POST_RETAILER_DEACTIVATED_ENTRY_PT:= \
	./cloud-functions/retailers/post_retailer_deactivate.go

POST_RETAILER_DEACTIVATED_FILES:= \
	./cloud-functions/retailers/models/retailer.go

PATCH_RETAILER_ENTRY_PT:= \
	./cloud-functions/retailers/patch_retailer.go

PATCH_RETAILER_FILES:= \
	./cloud-functions/retailers/models/retailer.go

#Site Package information
GET_SITE_ENTRY_PT:= \
	./cloud-functions/sites/get_site.go

GET_SITE_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/sites/common/common.go

GET_SITE_AUDIT_ENTRY_PT:= \
	./cloud-functions/sites/get_site_audit.go

GET_SITE_AUDIT_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/sites/common/common.go

PATCH_SITE_ENTRY_PT:= \
	./cloud-functions/sites/patch_site.go

PATCH_SITE_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/sites/common/common.go

PATCH_SITE_STATUS_ENTRY_PT:= \
	./cloud-functions/sites/patch_site_status.go

PATCH_SITE_STATUS_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/sites/common/common.go

POST_SITE_ENTRY_PT:= \
	./cloud-functions/sites/post_site.go

POST_SITE_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/sites/common/common.go

GET_SITES_ENTRY_PT:= \
	./cloud-functions/sites/get_sites.go

GET_SITES_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/sites/common/common.go

GET_SITE_SPOKES_ENTRY_PT:= \
	./cloud-functions/sites/get_site_spokes.go

GET_SITE_SPOKES_FILES:= \
	./cloud-functions/sites/models/site.go \
	./cloud-functions/spokes/models/spoke.go \
	./cloud-functions/sites/common/common.go

#Spoke Package information

POST_SPOKE_ENTRY_PT:= \
    ./cloud-functions/spokes/post_spoke.go

POST_SPOKE_FILES:= \
    ./cloud-functions/spokes/models/spoke.go

GET_SPOKE_ENTRY_PT:= \
    ./cloud-functions/spokes/get_spoke.go

GET_SPOKE_FILES:= \
    ./cloud-functions/spokes/models/spoke.go \
    ./cloud-functions/spokes/common/common.go

PATCH_SPOKE_ATTACH_ENTRY_PT:= \
    ./cloud-functions/spokes/patch_spoke_attach.go

PATCH_SPOKE_ATTACH_FILES:= \
    ./cloud-functions/spokes/models/spoke.go

PATCH_SPOKE_DETACH_ENTRY_PT:= \
    ./cloud-functions/spokes/patch_spoke_detach.go

PATCH_SPOKE_DETACH_FILES:= \
    ./cloud-functions/spokes/models/spoke.go \
    ./cloud-functions/spokes/common/common.go

GET_SPOKES_ENTRY_PT:= \
    ./cloud-functions/spokes/get_spokes.go

GET_SPOKES_FILES:= \
    ./cloud-functions/spokes/models/spoke.go \
    ./cloud-functions/spokes/common/common.go


PACKAGE_LIST:=  \
	AUDIT_ENTRY_PT:AUDIT_FILES:site-info-svc-${VERSION}-audit-pusher.zip \
	GET_RETAILER_ENTRY_PT:GET_RETAILER_FILES:site-info-svc-${VERSION}-get-retailer.zip \
	POST_RETAILER_ENTRY_PT:POST_RETAILER_FILES:site-info-svc-${VERSION}-post-retailer.zip \
	GET_RETAILERS_ENTRY_PT:GET_RETAILERS_FILES:site-info-svc-${VERSION}-get-retailers.zip \
	GET_RETAILER_AUDIT_ENTRY_PT:GET_RETAILER_AUDIT_FILES:site-info-svc-${VERSION}-get-retailer-audit.zip \
	POST_RETAILER_DEACTIVATED_ENTRY_PT:POST_RETAILER_DEACTIVATED_FILES:site-info-svc-${VERSION}-retailer-deactivate.zip \
	PATCH_RETAILER_ENTRY_PT:PATCH_RETAILER_FILES:site-info-svc-${VERSION}-patch-retailer.zip \
	GET_SITE_ENTRY_PT:GET_SITE_FILES:site-info-svc-${VERSION}-get-site.zip \
	GET_SITE_AUDIT_ENTRY_PT:GET_SITE_AUDIT_FILES:site-info-svc-${VERSION}-get-site-audit.zip \
	PATCH_SITE_ENTRY_PT:PATCH_SITE_FILES:site-info-svc-${VERSION}-patch-site.zip \
	PATCH_SITE_STATUS_ENTRY_PT:PATCH_SITE_STATUS_FILES:site-info-svc-${VERSION}-patch-site-status.zip \
	POST_SITE_ENTRY_PT:POST_SITE_FILES:site-info-svc-${VERSION}-post-site.zip \
	GET_SITES_ENTRY_PT:GET_SITES_FILES:site-info-svc-${VERSION}-get-sites.zip \
	GET_SITE_SPOKES_ENTRY_PT:GET_SITE_SPOKES_FILES:site-info-svc-${VERSION}-get-site-spokes.zip \
	POST_SPOKE_ENTRY_PT:POST_SPOKE_FILES:site-info-svc-${VERSION}-post-spoke.zip \
	GET_SPOKE_ENTRY_PT:GET_SPOKE_FILES:site-info-svc-${VERSION}-get-spoke.zip \
	PATCH_SPOKE_ATTACH_ENTRY_PT:PATCH_SPOKE_ATTACH_FILES:site-info-svc-${VERSION}-patch-spoke-attach.zip \
	PATCH_SPOKE_DETACH_ENTRY_PT:PATCH_SPOKE_DETACH_FILES:site-info-svc-${VERSION}-patch-spoke-detach.zip \
	GET_SPOKES_ENTRY_PT:GET_SPOKES_FILES:site-info-svc-${VERSION}-get-spokes.zip


.PHONY: audit
audit:
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Linting code...'
	golangci-lint run -c .golangci.yaml

.PHONY: clean
clean:
	go version
	rm -rf target
	rm -rf vendor
	go mod tidy

.PHONY: test
test:
	go mod vendor
	go fmt ./...
	golangci-lint run ./... --out-${NO_FUTURE}format colored-line-number
	go build ./...
	go test -cover -race ./...

.PHONY: release
release:
	go mod vendor
	go fmt ./...
	go build ./...

.PHONY: build
build: clean release
	# General packaging guideline to create a package is to first copy all the common files
	# to the packaging location. Since these files are going to be common for every package
	# it would be a good idea to create every package and delete all package specific things
    # this allowing the reuse of common files for packaging and removing the need to copying common files everytime

	#Copying Vendor Files
	mkdir -p  ${PACKAGING_FOLDER}/${VENDOR_FOLDER} ; \
	cp -R ${VENDOR_FOLDER} ${PACKAGING_FOLDER}/ \

	#Copying Common files
	find ./common -type d -not  -exec mkdir -p  ${PACKAGING_FOLDER}/{} \;
	find ./common -type f -not -iname '*_test.go' -exec cp '{}' '${PACKAGING_FOLDER}/{}' \;

	#Copying Mod files
	for name in ${MOD_FILES}; do \
		cp -R ./$${name} ${PACKAGING_FOLDER} ; \
	done

	#Create the packages for all the files
	$(foreach package, ${PACKAGE_LIST}, $(call generate_zip_file  ,  ${$(word 1,$(subst :, ,${package}))} , \
 		${$(word 2,$(subst :, ,${package}))} , $(word 3,$(subst :, ,${package})) ))

	#Final cleanup to remove all common data
	rm -rf ./${PACKAGING_FOLDER}

define generate_zip_file

	for name in $(1); do \
		echo $${name} ; \
		cp  ./$${name} ${PACKAGING_FOLDER}/ ; \
    done


	for name in $(2); do \
		echo $${name} ; \
		mkdir -p  `dirname ${PACKAGING_FOLDER}/$${name}` ; \
        cp  ./$${name} ${PACKAGING_FOLDER}/$${name} ; \
	done

	#remove the previously created package if one already exists
	rm -rf ../../${TARGET_FOLDER}/$(strip $(3))
	cd ${PACKAGING_FOLDER} && \
	zip -r -q ../../${TARGET_FOLDER}/$(strip $(3)) . \

	#Cleanup existing data created for the previous package
	# Note: it is being assumed that all the files are being copied into cloud-functions path only
	rm -rf ./${PACKAGING_FOLDER}/cloud-functions

	#delete all go files copied at the root folder
	rm -f  ${PACKAGING_FOLDER}/*.go

endef

.PHONY: all
all: clean test build

.FORCE:

