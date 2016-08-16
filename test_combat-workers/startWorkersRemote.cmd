@echo off
set /p UserInput=Enter count: 
set /a Test=UserInput

set /p UserInput2=Enter an address (http://...): 
set /a Test2=UserInput2


FOR /L %%A IN (1,1,%UserInput%) DO (
  del /F /S /Q %%A
  timeout 1
  mkdir %%A
  copy ..\combat-worker.exe %%A\
  cd %%A && start combat-worker.exe %UserInput2% && cd ..
  ECHO %%A
)