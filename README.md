# authz-gateway
An HTTP authorization gateway that decouples policy enforcement from applications using pluggable authorization engines.

## Overview

`authz-gateway` is an Authorization Service that acts as a Policy Enforcement Point for HTTP applications, API gateways, 
and reverse proxies.

It receives an authenticated HTTP request, translates it into a normalized authorization request,
delegates the decision to a configured authorization engine, and returns an allow or deny response.

## Goals

- Centralized authorization logic outside application code
- Integrate with API gateways and reverse proxies such as Traefik.
- Support pluggable authorization engines such as OPA and Cerbos.
- 

[!NOTE]
This project is at an early stage of development. Its APIs and configuration format may change.