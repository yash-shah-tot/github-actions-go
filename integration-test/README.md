# Integration Testing
---

### Overview

Using Postman's [newman](https://www.npmjs.com/package/newman) as command line tool the integration test can be exicuted. 

---

### Prerequisites

- [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)
- [newman](https://www.npmjs.com/package/newman#getting-started)
---

### Documentation Links

[Postman strategy for API Testing](https://takeofftech.atlassian.net/wiki/spaces/~62d1af37dcf59ca4ad022068/pages/3871768581/Postman+Strategy+for+API+Testing)

---

### Folder structure in postman

```
|- Test Root
|   |- Retailer
|   |  |-V1
|   |       |- Create
|   |       |   |- Negative Test cases
|   |       |   |- Positive Test cases
|   |       |
|   |       |- Update
|   |       |- Get
|   |       |- Get All 
|   |       |- Delete
|   |       |- Retailer Lifecycle Validation
|   |
|   |- Sites
|   |- Spokes
|   |- Site-Info Life Cycle
```
---


### Run Using command line

```
newman run integration-test/SiteInfo-TestCases.postman_collection.json -e integration-test/newmanEnv.postman_environment.json  --global-var "BEARERTOKEN=<BEARERTOKEN_VALUE>"
```

Note: BEARERTOKEN_VALUE need to be updated to access the API services.

---


### Run Using github action

- Visit `actions` tab and find the workflow named `Integration test` [here](https://github.com/TakeoffTech/site-info-svc/actions/workflows/newman.yml)
- Click on run workflow. It will open a popup for branch selection, select a branch where updated test cases and collection is present.
- Click on `Run workflow` button to run the test case.

---

