#!/bin/bash

# API Test Examples for workmate service
# Usage: ./curl_examples.sh

BASE_URL="http://localhost:8080"

echo "🚀 Starting API tests..."

# 1. Create a new task
echo "📝 Creating new task..."
TASK_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/tasks")
TASK_ID=$(echo $TASK_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TASK_ID" ]; then
    echo "❌ Failed to create task"
    echo "Response: $TASK_RESPONSE"
    exit 1
fi

echo "✅ Task created with ID: $TASK_ID"

# 2. Add files to the task
echo "📁 Adding files to task..."
curl -s -X POST "$BASE_URL/api/v1/tasks/$TASK_ID/files" \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://i.pinimg.com/736x/81/8f/d0/818fd0c7b9b0ebca8c753828bcb0a71b.jpg",
      "https://psv4.userapi.com/s/v1/d2/A2XhIypFjc4NjEOaROKSCZXC9_2HLKIh4pXZL_3WAO6i9OssY_eB0TLPTvmHz6KJry7P8avuqSt1-M_I8hHZJMANGP3HWTsNXWOjNMmsK6vUA9q2ySQ8yyCSFsvNDAM-cq8gnefQFcTa/Rannetriasovye_amfibii_Vostochnoy_Evropy.pdf",
      "https://psv4.userapi.com/s/v1/d2/o_HuRDUxdMzM8saNariRuufcOk_rbiezPUbTrS7aOdg4USCkaOw2oUTQ7Alle6CDRlXUdhOIAdRkHRAMgEfPbB-dcm39D9vZaXckmTvjkX84S9f1bFl_8xOO8Qs0ck2JDNvLV7Mm8BkK/Kak_slushat_muzyku_Max_Bazhenov.pdf"
    ]
  }'

echo "✅ Files added to task"

# 3. Check task status
echo "📊 Checking task status..."
STATUS_RESPONSE=$(curl -s "$BASE_URL/api/v1/tasks/$TASK_ID")
echo "Status response: $STATUS_RESPONSE"

# 4. Wait for processing (poll status)
echo "⏳ Waiting for processing to complete..."
for i in {1..30}; do
    STATUS=$(curl -s "$BASE_URL/api/v1/tasks/$TASK_ID" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    echo "Status: $STATUS (attempt $i/30)"
    
    if [ "$STATUS" = "ready" ]; then
        echo "✅ Processing completed!"
        break
    elif [ "$STATUS" = "failed" ]; then
        echo "❌ Processing failed!"
        break
    fi
    
    sleep 2
done

# 5. Download archive if ready
if [ "$STATUS" = "ready" ]; then
    echo "📦 Downloading archive..."
    curl -O "$BASE_URL/api/v1/tasks/$TASK_ID/archive"
    echo "✅ Archive downloaded as 'archive'"
else
    echo "⚠️  Task not ready for download (status: $STATUS)"
fi

echo "🎉 Test completed!"
