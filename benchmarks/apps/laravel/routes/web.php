<?php

// This benchmark app is API-only (routes/api.php); the health check is
// /livez (bootstrap/app.php's withRouting health:). No web routes — the
// scaffold's default `/` welcome-view route was removed along with the
// frontend resources.
