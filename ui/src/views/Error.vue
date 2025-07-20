<template>
  <div class="error-container">
    <el-icon :size="90" color="#f56565"><WarningIcon/></el-icon>
    <div class="error-content">
      <h2 class="error-title">{{ title }}</h2>
      <p class="error-message">{{ message }}</p>
    </div>
  </div>
</template>

<script setup>
import { Warning as WarningIcon } from '@vicons/ionicons5'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  err: String
})

const title = computed(() => {
  const err = props.err
  if (err === 'offline')
    return t('Device Unavailable')
  else if (err === 'full')
    return t('Terminal Session Limit Reached')
  else if (err === 'timeout')
    return t('Device Response Timeout')
  return ''
})

const message = computed(() => {
  const err = props.err
  if (err === 'offline')
    return t('The device is currently offline. Please check the device status and try again.')
  else if (err === 'full')
    return t('The maximum number of concurrent terminal sessions has been reached. Please try again later.')
  else if (err === 'timeout')
    return t('The device did not respond to the terminal session request within the expected time. Please check the device status and try again.')
  return ''
})
</script>

<style scoped>
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  text-align: center;
}

.error-content {
  max-width: 600px;
  animation: slideUp 0.8s ease-out 0.2s both;
}

.error-title {
  font-size: 1.8rem;
  font-weight: 600;
  color: #7a8fb0;
  margin-bottom: 1rem;
  line-height: 1.2;
}

.error-message {
  font-size: 1rem;
  color: #b6c1d3;
  margin-bottom: 2rem;
  line-height: 1.6;
  text-align: left;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: scale(0.8);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
