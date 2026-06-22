#!/bin/bash

envsubst < "/rabbitmq/definitions.tmpl.json" > "/rabbitmq/definitions.json"
