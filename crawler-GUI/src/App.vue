<script setup>
import { ref, onMounted } from 'vue'

const containers = ref([])
const seed = ref("")

// Load containers on mount
onMounted(() => {
  fetch("http://localhost:8080/containers")
      .then(res => res.json())
      .then(data => {
        console.log(data)
        containers.value = data
      })
      .catch(err => console.error(err))
})

// Function to stop worker container by id
function stopWorker(id) {
  fetch("http://localhost:8080/stop", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ container_id: id })
  })
      .then(res => res.json())
      .then(data => {
        console.log(data)
        containers.value = data
      })
      .catch(err => console.error(err))
}

// Function to start a new worker container
function startWorker() {
  fetch("http://localhost:8080/start", {
    headers: { "Content-Type": "application/json" }
  })
      .then(res => res.json())
      .then(data => {
        console.log(data)
        containers.value = data
      })
      .catch(err => console.error(err))
}

// Function to send seed data
function sendSeed() {
  fetch("http://localhost:8080/seed", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ seed: seed.value })
  })
      .then(res => res.json())
      .then(data => {
        console.log(data)
        containers.value = data
      })
      .catch(err => console.error(err))
}
</script>

<template>
  <h1>Crawlers dashboard</h1>
  <div>
    <p>Seed link</p>
    <input v-model="seed" type="text" />
    <button @click="sendSeed">Start</button>
  </div>
  <button @click="startWorker">Start new worker</button>
  <div id="container_out">
    <div id="container" v-for="container in containers" :key="container.id">
      <div id="container_inside">
        <p style="color: greenyellow">Running</p>
        <h3>{{ container.id }} ({{ container.image }})</h3>
        <button @click="stopWorker(container.id)">Stop</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
#container_out {
  display: flex;
  padding: 12px;
}

#container {
  padding: 12px;
}

#container_inside {
  border-radius: 16px;
  height: 200px;
  width: 300px;
  padding: 4px;
  background-color: #646cff;
}
</style>
