use axum::{
    routing::{get, post, delete, patch},
    Router,
    extract::{Path, State},
    Json,
};
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::sync::Arc;
use tokio::net::TcpListener;
use tracing::{info, error};
use std::time::Duration;

#[derive(Clone)]
struct AppState {
    incus_endpoint: String,
    client: reqwest::Client,
}

#[derive(Serialize, Deserialize)]
struct Instance {
    name: String,
    status: String,
    #[serde(rename = "type")]
    instance_type: String,
}

#[derive(Serialize, Deserialize)]
struct CreateInstanceRequest {
    name: String,
    source: serde_json::Value,
    #[serde(rename = "type")]
    instance_type: String,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();

    let state = Arc::new(AppState {
        incus_endpoint: "unix:///var/lib/incus/unix.socket".to_string(),
        client: reqwest::Client::builder()
            .timeout(Duration::from_secs(30))
            .build()?,
    });

    let app = Router::new()
        .route("/healthz", get(health_check))
        .route("/readyz", get(ready_check))
        .route("/api/v1/instances", get(list_instances).post(create_instance))
        .route("/api/v1/instances/:id", get(get_instance).delete(delete_instance).patch(update_instance))
        .route("/api/v1/storage-pools", get(list_storage_pools).post(create_storage_pool))
        .route("/api/v1/networks", get(list_networks).post(create_network))
        .route("/api/v1/profiles", get(list_profiles).post(create_profile))
        .with_state(state);

    let listener = TcpListener::bind("0.0.0.0:8080").await?;
    info!("API Gateway starting on :8080");
    axum::serve(listener, app).await?;

    Ok(())
}

async fn health_check() -> Json<serde_json::Value> {
    Json(json!({"status": "ok"}))
}

async fn ready_check(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match check_incus(state).await {
        Ok(_) => Ok(Json(json!({"status": "ready"}))),
        Err(_) => Err(axum::http::StatusCode::SERVICE_UNAVAILABLE),
    }
}

async fn check_incus(state: &AppState) -> anyhow::Result<()> {
    let resp = state.client.get("http://localhost/1.0")
        .send().await?;
    if resp.status().is_success() {
        Ok(())
    } else {
        Err(anyhow::anyhow!("Incus not ready"))
    }
}

async fn list_instances(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match fetch_incus(state, "/1.0/instances").await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to list instances: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn get_instance(State(state): State<Arc<AppState>>, Path(id): Path<String>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match fetch_incus(state, &format!("/1.0/instances/{}", id)).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to get instance: {}", e);
            Err(axum::http::StatusCode::NOT_FOUND)
        }
    }
}

async fn create_instance(State(state): State<Arc<AppState>>, Json(req): Json<serde_json::Value>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match post_incus(state, "/1.0/instances", req).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to create instance: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn delete_instance(State(state): State<Arc<AppState>>, Path(id): Path<String>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match delete_incus(state, &format!("/1.0/instances/{}", id)).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to delete instance: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn update_instance(State(state): State<Arc<AppState>>, Path(id): Path<String>, Json(req): Json<serde_json::Value>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match patch_incus(state, &format!("/1.0/instances/{}", id), req).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to update instance: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn list_storage_pools(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match fetch_incus(state, "/1.0/storage-pools").await {
        Ok(data) => Ok(Json(data)),
        Err(_) => Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR),
    }
}

async fn create_storage_pool(State(state): State<Arc<AppState>>, Json(req): Json<serde_json::Value>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match post_incus(state, "/1.0/storage-pools", req).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to create storage pool: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn list_networks(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match fetch_incus(state, "/1.0/networks").await {
        Ok(data) => Ok(Json(data)),
        Err(_) => Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR),
    }
}

async fn create_network(State(state): State<Arc<AppState>>, Json(req): Json<serde_json::Value>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match post_incus(state, "/1.0/networks", req).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to create network: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn list_profiles(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match fetch_incus(state, "/1.0/profiles").await {
        Ok(data) => Ok(Json(data)),
        Err(_) => Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR),
    }
}

async fn create_profile(State(state): State<Arc<AppState>>, Json(req): Json<serde_json::Value>) -> Result<Json<serde_json::Value>, axum::http::StatusCode> {
    match post_incus(state, "/1.0/profiles", req).await {
        Ok(data) => Ok(Json(data)),
        Err(e) => {
            error!("Failed to create profile: {}", e);
            Err(axum::http::StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn fetch_incus(state: &AppState, path: &str) -> anyhow::Result<serde_json::Value> {
    let resp = state.client.get(&format!("http://localhost{}", path))
        .send().await?;
    let data = resp.json().await?;
    Ok(data)
}

async fn post_incus(state: &AppState, path: &str, body: serde_json::Value) -> anyhow::Result<serde_json::Value> {
    let resp = state.client.post(&format!("http://localhost{}", path))
        .json(&body)
        .send().await?;
    let data = resp.json().await?;
    Ok(data)
}

async fn delete_incus(state: &AppState, path: &str) -> anyhow::Result<serde_json::Value> {
    let resp = state.client.delete(&format!("http://localhost{}", path))
        .send().await?;
    let data = resp.json().await?;
    Ok(data)
}

async fn patch_incus(state: &AppState, path: &str, body: serde_json::Value) -> anyhow::Result<serde_json::Value> {
    let resp = state.client.patch(&format!("http://localhost{}", path))
        .json(&body)
        .send().await?;
    let data = resp.json().await?;
    Ok(data)
}
