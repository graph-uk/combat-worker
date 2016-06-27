taskkill /im Combat-Worker.exe
timeout 1
del /F /S /Q 1
del /F /S /Q 2
del /F /S /Q 3
del /F /S /Q 4
del /F /S /Q 5
del /F /S /Q 6
del /F /S /Q 7
del /F /S /Q 8
timeout 1
mkdir 1
copy ..\combat-worker.exe 1\
timeout 1
xcopy 1 2\ /s /e /h /r
xcopy 1 3\ /s /e /h /r
xcopy 1 4\ /s /e /h /r
xcopy 1 5\ /s /e /h /r
xcopy 1 6\ /s /e /h /r
xcopy 1 7\ /s /e /h /r
xcopy 1 8\ /s /e /h /r
timeout 1
cd 1 && start combat-worker.exe http://localhost:9090 && cd ..
cd 2 && start combat-worker.exe http://localhost:9090 && cd ..
cd 3 && start combat-worker.exe http://localhost:9090 && cd ..
cd 4 && start combat-worker.exe http://localhost:9090 && cd ..
cd 5 && start combat-worker.exe http://localhost:9090 && cd ..
cd 6 && start combat-worker.exe http://localhost:9090 && cd ..
cd 7 && start combat-worker.exe http://localhost:9090 && cd ..
cd 8 && start combat-worker.exe http://localhost:9090 && cd ..