<template>
  <div class="stat-card" :style="{ '--accent': color }">
    <i :class="['pi', icon, 'stat-icon']" />
    <div class="stat-body">
      <span class="stat-value">{{ value ?? '-' }}</span>
      <span class="stat-label">{{ label }}</span>
    </div>
  </div>
</template>

<script setup>
defineProps({
  label: String,
  value: [Number, String],
  icon: String,
  color: { type: String, default: 'var(--primary)' },
})
</script>

<style scoped>
.stat-card {
  background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 1.25rem 1.5rem;
  box-shadow: 0 1px 4px rgba(0,0,0,.08);
  display: flex; align-items: center; gap: 1rem;
  border-left: 4px solid var(--accent);
}
.stat-icon { font-size: 1.5rem; color: var(--accent); }
.stat-body { display: flex; flex-direction: column; }
.stat-value { font-size: 1.75rem; font-weight: 700; line-height: 1; }
.stat-label { font-size: .8rem; color: var(--text-muted); margin-top: .25rem; }
</style>
