# site-info-svc

---

### Overview
This services provides the functionality to create Retailers, Sites and Spokes.
It stores all the data related to the retailer, sites and spokes.
It provides the user to create, update, delete, get and list the above entities.
This service is a multi-tenant solution and only one service shall cater to all the tenants.

---

### Prerequisites

- [go](https://go.dev/doc/install)
- [golangci-lint](https://golangci-lint.run/usage/install/)
---

### Documentation Links
[Site Info Service Design Document](https://takeofftech.atlassian.net/wiki/spaces/INC/pages/3806396520/Site+Information+Microservice+-+Design+Document)

---

### How do I build this project ?
```
make build
```
The build artifacts will be present in the target folder.
Multiple zips would be created which can be uploaded to the cloud function directly

---

### How do I test the code ?

#### Unit Testing
```
make clean test
```
This will run all the unit test cases and also check the code linting

#### Integration Testing
You can run the postman collection on the cloud functions deployed in your sandbox project
Detailed procedure given [here](https://github.com/TakeoffTech/site-info-svc/tree/main/integration-test/README.md)

---

### How do I Configure Linting ?
You will have to install golangci-lint in your local machine so that linting works
Please follow this [link](https://golangci-lint.run/usage/install/) for golangci-lint installation

To list down the linting issues in the project, run the below command
```
golangci-lint run -c .golangci.yaml
```
---
### How do I deploy this project in my sandbox ?

The recommended way to deploy this project is via terraform.
This project haa a deployable terraform project 
[site-info-svc-tf](https://github.com/TakeoffTech/site-info-svc-tf)

Copy the build artifacts (zips in target folder) to the src/artifacts folder in the terraform project (by default this folder does not exists).
Please follow the instructions in the site-info-svc-tf project to deploy the complete project service and its related resources

**Note : Make sure the GCP project that you are using should have the following service/API enabled**
- Firestore
- Cloud Trace
- Pub Sub
- Secrets Manager
- Storage Bucket
- Cloud Function
- Cloud Run
- Any dependant services if required

---

### How do I run this project locally ?
There is main.go file in each of the cloud function entity, set the values according to your project related values

```
os.Setenv("FUNCTION_TARGET", "FUNCTION_TARGET")
os.Setenv("PROJECT_ID", "PROJECT_ID")
os.Setenv("OPENCENSUSX_PROJECT_ID", "PROJECT_ID")
os.Setenv("AUDIT_LOG_TOPIC", "AUDIT_LOG_TOPIC")
```
if any more env is required to be set please look into the terraform code for the particular cloud function

---

### APIGEE to Service Configs

The service is accessible via apigee and the configurations can be found in the repo
[apigee-apis](https://github.com/TakeoffTech/apigee-apis)

---

