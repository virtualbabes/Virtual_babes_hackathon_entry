@echo off
:: ============================================================
:: setup_bot_pathway.bat  —  Bot/Pet pathway local-LLM builder
:: WRAPPER around the owner's setup_custom_quant_ornith.bat
:: Feeds a CURATED BEHAVIORAL corpus (local_bots/corpus/<pathway>/)
:: into the existing ornith imatrix + server harness.
:: The owner's harness (setup_custom_quant_ornith.bat) is NOT modified.
::
:: §31.1 — identity / tier / backend / port model:
::   Env (set by local_model_promotion.go, or manually):
::     ORNITH_IDENTITY      = unique citizen/pet wallet (namespaces the runner)
::     ORNITH_PATHWAY       = citizen|career|governor|sovereign|underworld|justice|breeder|family|pet_generic|pet_immature
::     ORNITH_MODEL_TIER    = tiny|small|base   (scalable AI power figure)
::     ORNITH_BACKEND       = gpu|cpu|auto       (auto-detected advisory cap)
::     ORNITH_DERIVED_PORT  = free port (base 11434 + hash(identity)%1000)
::     ORNITH_OLLAMA_GATEWAY= 1 -> skip standalone server, register Ollama alias ornith-<identity>
::
:: NO hard cap is enforced — the hardware scan only RECOMMENDS a max instance count.
:: ============================================================
setlocal EnableExtensions
title Bot Pathways — Ornith Corpus Builder (wrapper)
color 0B
cls

:: Resolve pathway: CLI arg OR env ORNITH_PATHWAY.
if not "%~1"=="" (
    set PATHWAY=%~1
) else if defined ORNITH_PATHWAY (
    set PATHWAY=%ORNITH_PATHWAY%
) else (
    echo USAGE: setup_bot_pathway.bat ^<pathway^>
    echo   OR set ORNITH_PATHWAY env. pathway = citizen ^| career ^| governor ^| sovereign ^| underworld ^| justice ^| breeder ^| family ^| pet_generic ^| pet_immature
    pause
    exit /b 1
)
if not defined ORNITH_IDENTITY set "ORNITH_IDENTITY=%PATHWAY%"

set REPO_DIR=Z:\Crypto_Draught\NFT-Seduction
set CORPUS_DIR=%REPO_DIR%\local_bots\corpus
set HARNESS=%REPO_DIR%\setup_custom_quant_ornith.bat
set TEMP_OUT_DIR=%USERPROFILE%\AppData\Local\Temp\zap_matrix_build
set PATHWAY_CALIB=%TEMP_OUT_DIR%\training_code_%PATHWAY%.txt
set MODELS_DIR=%USERPROFILE%\OneDrive\Desktop\models

if not exist "%HARNESS%" (
    echo CRITICAL ERROR: owner harness not found at %HARNESS%
    pause
    exit /b 1
)

:: Locate the pathway corpus (Users/Bots/Pets subtrees)
set FOUND=0
for %%T in (Users Bots Pets) do (
    if exist "%CORPUS_DIR%\%%T\%PATHWAY%\README.md" (
        set CORPUS_SUB=%%T\%PATHWAY%
        set FOUND=1
    )
)
if "%FOUND%"=="0" (
    echo ERROR: no corpus for pathway "%PATHWAY%" under %CORPUS_DIR%\{Users,Bots,Pets}\
    pause
    exit /b 1
)

echo [BOT-PATHWAY] Building calibration for pathway: %PATHWAY%  (corpus: %CORPUS_SUB%)
echo [BOT-PATHWAY] Identity=%ORNITH_IDENTITY%  Tier=%ORNITH_MODEL_TIER%  Backend=%ORNITH_BACKEND%  Port=%ORNITH_DERIVED_PORT%
if not exist "%TEMP_OUT_DIR%" mkdir "%TEMP_OUT_DIR%"

:: Concatenate the pathway corpus (README + any extra .md in the dir) into the calibration file.
echo --- BOT_PATHWAY_CALIBRATION: %PATHWAY% --- > "%PATHWAY_CALIB%"
for /r "%CORPUS_DIR%\%CORPUS_SUB%" %%f in (*.md) do (
    echo --- PATHWAY_DOC_START: %%~nxf ^| PATH: %%f --- >> "%PATHWAY_CALIB%"
    type "%%f" >> "%PATHWAY_CALIB%" 2>nul
    echo. >> "%PATHWAY_CALIB%"
)
echo [BOT-PATHWAY] Corpus concatenated -> %PATHWAY_CALIB%

:: The owner's harness reads its OWN expected calibration path (%TEMP_OUT_DIR%\training_code.txt)
:: and SKIPS its workspace snapshot when that file exists. We place our corpus there so the harness
:: uses the behavioral corpus instead of the whole-repo source dump — WITHOUT editing the owner's .bat.
copy /y "%PATHWAY_CALIB%" "%TEMP_OUT_DIR%\training_code.txt" >nul

echo [BOT-PATHWAY] Invoking owner harness: %HARNESS%
call "%HARNESS%"

:: ---- §31.1 POST-STEP: namespace the emitted runner (owns the file surgery; owner .bat untouched) ----
set EMITTED=%MODELS_DIR%\start_ornith_matrix.bat
if exist "%EMITTED%" (
    if "%ORNITH_OLLAMA_GATEWAY%"=="1" (
        echo [BOT-PATHWAY] Ollama-gateway mode: alias ornith-%ORNITH_IDENTITY% registered by harness; skipping port rewrite.
    ) else (
        set NAMESPACED=%MODELS_DIR%\start_ornith_%ORNITH_IDENTITY%.bat
        copy /y "%EMITTED%" "%NAMESPACED%" >nul
        :: Rewrite the fixed 11434 port to the derived free port so identities don't collide.
        if defined ORNITH_DERIVED_PORT (
            powershell -NoProfile -Command "(Get-Content '%NAMESPACED%') -replace '11434', '%ORNITH_DERIVED_PORT%' | Set-Content '%NAMESPACED%'"
        )
        echo [BOT-PATHWAY] Namespaced runner -> %NAMESPACED%  (port %ORNITH_DERIVED_PORT%)
    )
)

echo ============================================================
echo  PATHWAY %PATHWAY% build delegated to owner harness.
echo  Runner (if produced) lands in the owner's model dir.
echo ============================================================
endlocal
pause
