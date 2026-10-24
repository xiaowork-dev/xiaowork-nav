<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="page">
    <section class="panel notfound">
      <h1>404</h1>
      <p class="muted">页面不存在。</p>
      <RouterLink class="badge" to="/">返回首页</RouterLink>
    </section>
  </div>
</template>

<style scoped>
.notfound {
  max-width: 420px;
  margin: 15vh auto 0;
  padding: 40px 28px;
  text-align: center;
}

.notfound h1 {
  margin: 0 0 8px;
  font-size: 3rem;
  font-weight: 800;
}
</style>
