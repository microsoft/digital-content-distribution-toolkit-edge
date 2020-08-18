#!/bin/bash
cp $1 Dockerfile_tmp; sed -i "s/adb2c_client_id/$ADB2C_CLIENT_ID/g" Dockerfile_tmp; sed -i "s/adb2c_client_secret/$ADB2C_CLIENT_SECRET/g" Dockerfile_tmp; sed -i "s/adb2c_tenant_id/$ADB2C_TENANT_ID/g" Dockerfile_tmp; sed -i "s/key_vault_name/$KEY_VAULT_NAME/g" Dockerfile_tmp; sed -i "s/b2c_app_tenant_id_val/$b2c_app_tenant_id/g" Dockerfile_tmp; sed -i "s/CLIENT_ID_val/$CLIENT_ID/g" Dockerfile_tmp; sed -i "s/HUB_CRM_API_KEY_val/$HUB_CRM_API_KEY/g" Dockerfile_tmp; cat Dockerfile_tmp;