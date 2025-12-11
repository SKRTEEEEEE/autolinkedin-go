# Test completo del flujo de Fase 2
$baseUrl = "http://localhost:8080"
$userId = "000000000000000000000001"

Write-Host "`n=== FASE 2 - TEST COMPLETO ===" -ForegroundColor Cyan

# Paso 1: Obtener ideas disponibles
Write-Host "`n[1/5] Obteniendo ideas disponibles..." -ForegroundColor Yellow
try {
    $ideasResponse = Invoke-RestMethod -Uri "$baseUrl/v1/ideas/$userId?limit=10" -Method Get
    Write-Host "Ideas encontradas: $($ideasResponse.ideas.Count)" -ForegroundColor Green
} catch {
    Write-Host "ERROR al obtener ideas: $_" -ForegroundColor Red
    exit 1
}

# Buscar una idea no usada
$unusedIdea = $ideasResponse.ideas | Where-Object { $_.used -eq $false } | Select-Object -First 1

if (-not $unusedIdea) {
    Write-Host "ERROR: No hay ideas sin usar disponibles" -ForegroundColor Red
    exit 1
}

$ideaId = $unusedIdea.id
Write-Host "Idea seleccionada: $ideaId" -ForegroundColor Green
Write-Host "Contenido: $($unusedIdea.content.Substring(0, [Math]::Min(100, $unusedIdea.content.Length)))..." -ForegroundColor Gray

# Paso 2: Generar drafts
Write-Host "`n[2/5] Generando drafts..." -ForegroundColor Yellow
$generateBody = @{
    user_id = $userId
    idea_id = $ideaId
} | ConvertTo-Json

try {
    $generateResponse = Invoke-RestMethod -Uri "$baseUrl/v1/drafts/generate" -Method Post -Body $generateBody -ContentType "application/json"
} catch {
    Write-Host "ERROR al generar drafts: $_" -ForegroundColor Red
    Write-Host "Response: $($_.Exception.Response | ConvertTo-Json)" -ForegroundColor Red
    exit 1
}
$jobId = $generateResponse.job_id
Write-Host "Job creado: $jobId" -ForegroundColor Green

# Paso 3: Consultar estado del job (polling)
Write-Host "`n[3/5] Consultando estado del job..." -ForegroundColor Yellow
$maxAttempts = 30
$attempt = 0
$jobCompleted = $false

while ($attempt -lt $maxAttempts -and -not $jobCompleted) {
    $attempt++
    Write-Host "  Intento $attempt/$maxAttempts..." -NoNewline
    
    try {
        $jobStatus = Invoke-RestMethod -Uri "$baseUrl/v1/drafts/jobs/$jobId" -Method Get -ContentType "application/json"
        Write-Host " Status: $($jobStatus.status)" -ForegroundColor Cyan
        
        if ($jobStatus.status -eq "completed") {
            $jobCompleted = $true
            Write-Host "Job completado exitosamente!" -ForegroundColor Green
            Write-Host "Drafts generados: $($jobStatus.draft_ids.Count)" -ForegroundColor Green
        } elseif ($jobStatus.status -eq "failed") {
            Write-Host "ERROR: Job failed - $($jobStatus.error)" -ForegroundColor Red
            exit 1
        } else {
            Start-Sleep -Seconds 2
        }
    } catch {
        Write-Host " Error: $_" -ForegroundColor Red
        Start-Sleep -Seconds 2
    }
}

if (-not $jobCompleted) {
    Write-Host "TIMEOUT: Job no completado después de $maxAttempts intentos" -ForegroundColor Red
    exit 1
}

# Paso 4: Verificar drafts generados
Write-Host "`n[4/5] Verificando drafts generados..." -ForegroundColor Yellow
$draftsResponse = Invoke-RestMethod -Uri "$baseUrl/v1/drafts/$userId?status=DRAFT" -Method Get -ContentType "application/json"
Write-Host "Drafts totales: $($draftsResponse.count)" -ForegroundColor Green

# Contar posts y artículos
$posts = $draftsResponse.drafts | Where-Object { $_.type -eq "POST" }
$articles = $draftsResponse.drafts | Where-Object { $_.type -eq "ARTICLE" }

Write-Host "  - Posts: $($posts.Count)" -ForegroundColor Cyan
Write-Host "  - Artículos: $($articles.Count)" -ForegroundColor Cyan

# Paso 5: Verificar que la idea fue marcada como usada
Write-Host "`n[5/5] Verificando que la idea fue marcada como usada..." -ForegroundColor Yellow
$updatedIdeasResponse = Invoke-RestMethod -Uri "$baseUrl/v1/ideas/$userId" -Method Get -ContentType "application/json"
$usedIdea = $updatedIdeasResponse.ideas | Where-Object { $_.id -eq $ideaId } | Select-Object -First 1

if ($usedIdea.used -eq $true) {
    Write-Host "Idea marcada como usada correctamente" -ForegroundColor Green
} else {
    Write-Host "ADVERTENCIA: Idea no fue marcada como usada" -ForegroundColor Yellow
}

# Resumen final
Write-Host "`n=== RESUMEN ===" -ForegroundColor Cyan
Write-Host "Job ID: $jobId" -ForegroundColor White
Write-Host "Idea ID: $ideaId" -ForegroundColor White
Write-Host "Estado final: $($jobStatus.status)" -ForegroundColor White
Write-Host "Drafts generados: $($draftsResponse.count)" -ForegroundColor White
Write-Host "  - Posts: $($posts.Count)/5" -ForegroundColor White
Write-Host "  - Artículos: $($articles.Count)/1" -ForegroundColor White

if ($posts.Count -eq 5 -and $articles.Count -eq 1) {
    Write-Host "`n✅ FLUJO COMPLETADO EXITOSAMENTE!" -ForegroundColor Green
} else {
    Write-Host "`n⚠️ FLUJO COMPLETADO CON ADVERTENCIAS" -ForegroundColor Yellow
    Write-Host "Se esperaban 5 posts y 1 artículo" -ForegroundColor Yellow
}
