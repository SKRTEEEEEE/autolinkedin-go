# Test simple del flujo de Fase 2
$baseUrl = "http://localhost:8080"
$userId = "000000000000000000000001"
$headers = @{"Content-Type" = "application/json"}

Write-Host "`n=== FASE 2 - TEST SIMPLE ===" -ForegroundColor Cyan

# Paso 1: Obtener ideas
Write-Host "`n[1/5] Obteniendo ideas..." -ForegroundColor Yellow
$response = Invoke-WebRequest -Uri "$baseUrl/v1/ideas/$userId" -Method Get -Headers $headers
$ideas = ($response.Content | ConvertFrom-Json).ideas
Write-Host "Ideas: $($ideas.Count)" -ForegroundColor Green

# Buscar idea no usada
$idea = $ideas | Where-Object { $_.used -eq $false } | Select-Object -First 1
if (-not $idea) {
    Write-Host "ERROR: No hay ideas sin usar" -ForegroundColor Red
    exit 1
}
Write-Host "Idea: $($idea.id)" -ForegroundColor Green
Write-Host "Contenido: $($idea.content.Substring(0, 80))..." -ForegroundColor Gray

# Paso 2: Generar drafts
Write-Host "`n[2/5] Generando drafts..." -ForegroundColor Yellow
$body = @{
    user_id = $userId
    idea_id = $idea.id
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "$baseUrl/v1/drafts/generate" -Method Post -Body $body -Headers $headers
$jobData = $response.Content | ConvertFrom-Json
$jobId = $jobData.job_id
Write-Host "Job ID: $jobId" -ForegroundColor Green

# Paso 3: Esperar y consultar estado
Write-Host "`n[3/5] Esperando completaci\u00f3n..." -ForegroundColor Yellow
$maxAttempts = 20
for ($i = 1; $i -le $maxAttempts; $i++) {
    Start-Sleep -Seconds 3
    Write-Host "  Intento $i/$maxAttempts..." -NoNewline
    
    $response = Invoke-WebRequest -Uri "$baseUrl/v1/drafts/jobs/$jobId" -Method Get -Headers $headers
    $job = $response.Content | ConvertFrom-Json
    
    Write-Host " $($job.status)" -ForegroundColor Cyan
    
    if ($job.status -eq "completed") {
        Write-Host "  Completado! Drafts: $($job.draft_ids.Count)" -ForegroundColor Green
        break
    } elseif ($job.status -eq "failed") {
        Write-Host "  ERROR: $($job.error)" -ForegroundColor Red
        exit 1
    }
}

if ($job.status -ne "completed") {
    Write-Host "TIMEOUT" -ForegroundColor Red
    exit 1
}

# Paso 4: Verificar drafts
Write-Host "`n[4/5] Verificando drafts..." -ForegroundColor Yellow
$response = Invoke-WebRequest -Uri "$baseUrl/v1/drafts/$userId" -Method Get -Headers $headers
$drafts = ($response.Content | ConvertFrom-Json).drafts
$posts = $drafts | Where-Object { $_.type -eq "POST" }
$articles = $drafts | Where-Object { $_.type -eq "ARTICLE" }

Write-Host "Total drafts: $($drafts.Count)" -ForegroundColor Green
Write-Host "  Posts: $($posts.Count)" -ForegroundColor Cyan  
Write-Host "  Artículos: $($articles.Count)" -ForegroundColor Cyan

# Paso 5: Verificar idea usada
Write-Host "`n[5/5] Verificando idea usada..." -ForegroundColor Yellow
$response = Invoke-WebRequest -Uri "$baseUrl/v1/ideas/$userId" -Method Get -Headers $headers
$ideas = ($response.Content | ConvertFrom-Json).ideas
$usedIdea = $ideas | Where-Object { $_.id -eq $idea.id }

if ($usedIdea.used) {
    Write-Host "Idea marcada como usada" -ForegroundColor Green
} else {
    Write-Host "ADVERTENCIA: Idea NO marcada" -ForegroundColor Yellow
}

# Resumen
Write-Host "`n=== RESUMEN ===" -ForegroundColor Cyan
Write-Host "Job: $jobId" -ForegroundColor White
Write-Host "Idea: $($idea.id)" -ForegroundColor White  
Write-Host "Drafts: $($drafts.Count) (Posts: $($posts.Count), Artículos: $($articles.Count))" -ForegroundColor White

if ($posts.Count -ge 5 -and $articles.Count -ge 1) {
    Write-Host "`n✅ ÉXITO COMPLETO!" -ForegroundColor Green
} else {
    Write-Host "`n⚠️  Advertencia: Se esperaban 5 posts y 1 artículo" -ForegroundColor Yellow
}
