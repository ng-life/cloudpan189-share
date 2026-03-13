<template>
  <n-modal
    v-model:show="showModal"
    preset="dialog"
    :title="props.updateMode ? '更新扫码令牌' : '扫码登录'"
    :mask-closable="false"
  >
    <div class="qrcode-login-container">
      <!-- 二维码显示区域 -->
      <div class="qrcode-section">
        <div v-if="loading" class="loading-container">
          <n-spin size="large" />
          <p>正在生成二维码...</p>
        </div>

        <div v-else-if="qrcodeUrl" class="qrcode-container">
          <n-qr-code :value="qrcodeUrl" :size="200" style="width: 220px; height: 220px" />
          <div class="qrcode-info">
            <n-icon size="20" color="#18a058">
              <CheckmarkCircleOutline />
            </n-icon>
            <span>二维码已生成，等待扫码中...</span>
          </div>
        </div>

        <div v-else class="error-container">
          <n-icon size="40" color="#d03050">
            <CloseCircleOutline />
          </n-icon>
          <p>二维码生成失败</p>
          <n-button type="primary" @click="initQrcode">重新生成</n-button>
        </div>
      </div>

      <!-- 操作说明 -->
      <div class="instructions">
        <h4>扫码登录步骤：</h4>
        <ol>
          <li>打开天翼云盘APP</li>
          <li>点击右上角"扫一扫"</li>
          <li>扫描上方二维码</li>
          <li>在APP中确认登录，页面将自动跳转</li>
        </ol>

        <!-- 倒计时显示 -->
        <div v-if="countdown > 0" class="countdown">
          <n-icon size="16" color="#f0a020">
            <TimeOutline />
          </n-icon>
          <span>二维码将在 {{ countdown }} 秒后过期</span>
        </div>

        <div v-else-if="qrcodeUrl" class="expired">
          <n-icon size="16" color="#d03050">
            <CloseCircleOutline />
          </n-icon>
          <span>二维码已过期</span>
          <n-button type="primary" size="small" @click="initQrcode" style="margin-left: 8px">
            重新生成
          </n-button>
        </div>
      </div>

      <!-- 状态提示 -->
      <div v-if="statusMessage" class="status-message">
        <n-alert :type="statusType" :title="statusMessage" />
      </div>
    </div>

    <template #action>
      <n-space>
        <n-button @click="handleCancel">取消</n-button>
        <n-button type="primary" @click="initQrcode" :loading="loading"> 重新生成二维码 </n-button>
        <n-button
          v-if="qrcodeUrl && countdown > 0"
          type="success"
          @click="handleCheckLogin"
          :loading="checkLoading"
        >
          手动确认登录
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { NModal, NQrCode, NSpin, NButton, NIcon, NSpace, NAlert, useMessage } from 'naive-ui'
import { CheckmarkCircleOutline, CloseCircleOutline, TimeOutline } from '@vicons/ionicons5'
import { initQrcode as initQrcodeApi, checkQrcode } from '@/api/cloudtoken'

// Props
interface Props {
  show: boolean
  updateMode?: boolean // 是否为更新模式
  tokenId?: number // 更新时的令牌ID
}

// Emits
interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

/** 轮询间隔（毫秒） */
const POLLING_INTERVAL_MS = 2000
/** 连续网络错误阈值，超过后停止轮询并提示 */
const MAX_CONSECUTIVE_ERRORS = 3

// 响应式数据
const loading = ref(false)
const checkLoading = ref(false)
const qrcodeUuid = ref('')
const qrcodeUrl = ref('')
const countdown = ref(0)
const statusMessage = ref('')
const statusType = ref<'success' | 'info' | 'warning' | 'error'>('info')

// 定时器
let countdownTimer: number | null = null
let pollingTimer: number | null = null
let isPolling = false // 防止并发轮询请求
let consecutiveErrors = 0 // 连续网络错误计数

// 消息提示
const message = useMessage()

// 计算属性
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value),
})

// 监听弹窗显示状态
watch(showModal, (newVal) => {
  if (newVal) {
    // 弹窗打开时初始化二维码
    initQrcode()
  } else {
    // 弹窗关闭时清理定时器
    clearTimers()
    resetState()
  }
})

// 初始化二维码
const initQrcode = () => {
  loading.value = true
  statusMessage.value = ''
  clearTimers()

  initQrcodeApi()
    .then((response) => {
      if (response.code === 200 && response.data) {
        qrcodeUuid.value = response.data.uuid
        // 二维码内容直接使用 UUID（参考 cloudpan189-interface README：
        // "生成二维码（二维码内容就是UUID）"），天翼云盘 APP 扫码后会识别该 UUID
        // 并调用 qrcodeLoginResult.action 完成确认
        qrcodeUrl.value = response.data.uuid

        // 开始倒计时（120秒）和自动轮询
        startCountdown(120)
        startPolling()

        statusMessage.value = '二维码生成成功，请使用天翼云盘APP扫描上方二维码并确认登录'
        statusType.value = 'success'
      } else {
        throw new Error(response.msg || '初始化二维码失败')
      }
    })
    .catch((error) => {
      console.error('初始化二维码失败:', error)
      statusMessage.value = '二维码生成失败，请重试'
      statusType.value = 'error'
      qrcodeUrl.value = ''
    })
    .finally(() => {
      loading.value = false
    })
}

// 开始倒计时
const startCountdown = (seconds: number) => {
  countdown.value = seconds

  countdownTimer = setInterval(() => {
    countdown.value--

    if (countdown.value <= 0) {
      clearInterval(countdownTimer!)
      countdownTimer = null
      stopPolling()
      statusMessage.value = '二维码已过期，请重新生成'
      statusType.value = 'warning'
    }
  }, 1000) as unknown as number
}

// 开始自动轮询（每 POLLING_INTERVAL_MS 毫秒检查一次扫码状态）
const startPolling = () => {
  stopPolling()
  pollingTimer = setInterval(() => {
    if (countdown.value <= 0 || !qrcodeUuid.value) {
      stopPolling()
      return
    }
    autoCheckLogin()
  }, POLLING_INTERVAL_MS) as unknown as number
}

// 停止自动轮询
const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
  isPolling = false
  consecutiveErrors = 0
}

// 自动轮询检查登录状态（静默，不打扰用户）
const autoCheckLogin = () => {
  if (isPolling || !qrcodeUuid.value) return
  isPolling = true

  const checkData = {
    uuid: qrcodeUuid.value,
    ...(props.updateMode && props.tokenId ? { id: props.tokenId } : {}),
  }

  checkQrcode(checkData)
    .then((response) => {
      consecutiveErrors = 0 // 请求成功，重置网络错误计数
      if (response.code === 200) {
        // 扫码登录成功
        handleLoginSuccess()
      } else if (response.code === 40003) {
        // 等待用户扫码确认，继续轮询（不做任何操作）
      } else if (response.code === 40001) {
        // 二维码已过期
        stopPolling()
        clearTimers()
        countdown.value = 0
        statusMessage.value = '二维码已过期，请重新生成'
        statusType.value = 'warning'
      } else {
        // 其他错误，停止轮询
        stopPolling()
        statusMessage.value = response.msg || '检查登录状态失败，请手动点击确认'
        statusType.value = 'error'
      }
    })
    .catch(() => {
      // 网络异常，计数并在连续失败过多时停止轮询并告知用户
      consecutiveErrors++
      if (consecutiveErrors >= MAX_CONSECUTIVE_ERRORS) {
        stopPolling()
        statusMessage.value = '网络连接异常，请检查网络后重试'
        statusType.value = 'error'
      }
    })
    .finally(() => {
      isPolling = false
    })
}

// 登录成功的统一处理
const handleLoginSuccess = () => {
  clearTimers()
  statusMessage.value = '登录成功！'
  statusType.value = 'success'
  message.success('扫码登录成功')

  setTimeout(() => {
    showModal.value = false
    emit('success')
  }, 1500)
}

// 手动检查登录状态（作为自动轮询的备用方式）
const handleCheckLogin = () => {
  if (!qrcodeUuid.value) {
    message.error('二维码ID不存在，请重新生成二维码')
    return
  }

  checkLoading.value = true

  const checkData = {
    uuid: qrcodeUuid.value,
    ...(props.updateMode && props.tokenId ? { id: props.tokenId } : {}),
  }

  checkQrcode(checkData)
    .then((response) => {
      if (response.code === 200) {
        // 登录成功
        handleLoginSuccess()
      } else if (response.code === 40001) {
        // 二维码已过期
        clearTimers()
        countdown.value = 0
        statusMessage.value = '二维码已过期，请重新生成'
        statusType.value = 'warning'
        message.warning('二维码已过期')
      } else if (response.code === 40002) {
        // 用户取消登录
        stopPolling()
        statusMessage.value = '用户取消了登录'
        statusType.value = 'info'
        message.info('用户取消了登录')
      } else if (response.code === 40003) {
        // 等待用户扫码
        statusMessage.value = '请先使用天翼云盘APP扫码并确认登录'
        statusType.value = 'warning'
        message.warning('请先扫码确认登录')
      } else {
        // 其他错误
        message.error(response.msg || '检查登录状态失败')
      }
    })
    .catch((error) => {
      console.error('检查二维码状态失败:', error)
      message.error('检查登录状态失败')
    })
    .finally(() => {
      checkLoading.value = false
    })
}

// 清理所有定时器
const clearTimers = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  stopPolling()
}

// 重置状态
const resetState = () => {
  qrcodeUuid.value = ''
  qrcodeUrl.value = ''
  countdown.value = 0
  statusMessage.value = ''
  loading.value = false
  checkLoading.value = false
}

// 取消操作
const handleCancel = () => {
  showModal.value = false
}

// 组件卸载时清理定时器
onUnmounted(() => {
  clearTimers()
})
</script>

<style scoped>
.qrcode-login-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
  min-height: 400px;
}

.qrcode-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 40px;
}

.qrcode-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.qrcode-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #18a058;
  font-size: 14px;
}

.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 40px;
}

.instructions {
  width: 100%;
  max-width: 300px;
  text-align: left;
}

.instructions h4 {
  margin: 0 0 12px;
  color: #333;
  font-size: 16px;
}

.instructions ol {
  margin: 0 0 16px;
  padding-left: 20px;
}

.instructions li {
  margin-bottom: 4px;
  color: #666;
  font-size: 14px;
}

.countdown {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #f0a020;
  font-size: 14px;
  margin-top: 12px;
}

.expired {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #d03050;
  font-size: 14px;
  margin-top: 12px;
}

.status-message {
  width: 100%;
  margin-top: 16px;
}
</style>
