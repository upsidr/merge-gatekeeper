.PHONY: build
build: 
	GO111MODULE=on LANG=en_US.UTF-8 CGO_ENABLED=0 go build ./cmd/merge-gatekeeper

.PHONY: docker-build
docker-build:
	docker build -t merge-gatekeeper:latest .

.PHONY: test
test:
	go test ./...

define IGNORED_JOBS
ci/circleci: deploy-acuity-model-service/Approve UAT Acuity Model Service,
ci/circleci: deploy-acuity-model-service/Approve QA Acuity Model Service,
ci/circleci: deploy-audit-service/Approve QA Audit Service,
ci/circleci: deploy-audit-service/Approve UAT Audit Service,
ci/circleci: deploy-logistics-service/Approve UAT Logistics Service,
ci/circleci: deploy-logistics-service/Approve Reset UAT Logistics Service DB,
ci/circleci: deploy-logistics-service/Approve QA Logistics Service,
ci/circleci: deploy-clinicalkpi-service/Approve QA ClinicalKPI Service,
ci/circleci: deploy-clinicalkpi-service/Approve UAT ClinicalKPI Service,
ci/circleci: deploy-logistics-service/Approve Reset QA Logistics Service DB,
ci/circleci: deploy-pophealth-service/Approve QA Pophealth Service,
ci/circleci: deploy-pophealth-service/Approve UAT Pophealth Service,
ci/circleci: deploy-patients-service/Approve UAT Patients Service,
ci/circleci: deploy-pophealth-service/Approve Reset QA Pophealth Service DB,
ci/circleci: deploy-patients-service/Approve QA Patients Service,
ci/circleci: deploy-pophealth-service/Approve Reset UAT Pophealth Service DB,
ci/circleci: deploy-caremanager-service/Approve QA CareManager Service,
ci/circleci: deploy-caremanager-service/Approve UAT CareManager Service,
ci/circleci: deploy-logistics-optimizer-service/Approve UAT Logistics Optimizer Service,
ci/circleci: deploy-caremanager-service/Approve Prod CareManager Service,
ci/circleci: deploy-logistics-optimizer-service/Approve QA Logistics Optimizer Service,
ci/circleci: deploy-tytocare-service/Approve QA TytoCare Service,
ci/circleci: deploy-tytocare-service/Approve UAT TytoCare Service,
ci/circleci: deploy-acuity-model-service/Approve Prod Acuity Model Service,
ci/circleci: deploy-audit-service/Approve Prod Audit Service,
ci/circleci: deploy-logistics-service/Approve Prod Logistics Service,
ci/circleci: deploy-patients-service/Approve Prod Patients Service,
ci/circleci: deploy-logistics-optimizer-service/Approve Prod Logistics Optimizer Service,
ci/circleci: deploy-tytocare-service/Approve Prod TytoCare Service,
ci/circleci: publish_npm_packages/Approve Publishing @dispatchhealth/nest-datadog,
Release / Build: services
endef
export IGNORED_JOBS
REPO="DispatchHealth/services"
REF="refs/heads/trunk"
.PHONY: run
run: build
	./merge-gatekeeper validate --ref ${REF} --token ${GITHUB_TOKEN} -r ${REPO} --ignored "$${IGNORED_JOBS}"