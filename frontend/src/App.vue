<template>
  <div class="app-root dark">
    <Toast position="top-right" />
    <ConfirmDialog />
    <RouterView />
  </div>
</template>

<script setup>
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
</script>

<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
</style>
