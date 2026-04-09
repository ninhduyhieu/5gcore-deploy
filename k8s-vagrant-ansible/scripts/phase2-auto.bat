@echo off
setlocal

echo [PHASE2] Build images on master...
vagrant provision k8s-master --provision-with phase2-build
if errorlevel 1 exit /b 1

echo [PHASE2] Load images into worker1...
vagrant provision k8s-worker1 --provision-with phase2-load
if errorlevel 1 exit /b 1

echo [PHASE2] Load images into worker2...
vagrant provision k8s-worker2 --provision-with phase2-load
if errorlevel 1 exit /b 1

echo [PHASE2] Apply manifests on master...
vagrant provision k8s-master --provision-with phase2-apply
if errorlevel 1 exit /b 1

echo [PHASE2 DONE]
exit /b 0