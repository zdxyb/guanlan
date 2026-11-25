<template>
  <NConfigProvider class="h-full" :theme="theme" :locale="uiLocale">
    <NaiveProvider>
      <IssuerView v-if="isIssuer"/>
      <RouterView v-else/>
      <NModal
          :show="showLicense && !isIssuer"
          preset="dialog"
          :title="t('license.title')"
          :show-icon="false"
          :mask-closable="false"
          :close-on-esc="false"
          :closable="false"
          style="--wails-draggable:no-drag"
      >
        <div class="text-sm leading-6 text-gray-500">{{ t('license.tip') }}</div>
        <div class="mt-3 flex items-center justify-between gap-3 rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-800">
          <div class="min-w-0">
            <div class="text-xs text-slate-500">{{ t('license.machine') }}</div>
            <div class="mt-1 truncate font-mono text-sm">{{ machineCode }}</div>
          </div>
          <NButton secondary size="small" @click="copyMachine">
            <template #icon>
              <NIcon><CopyOutline/></NIcon>
            </template>
            {{ t('license.copy_machine') }}
          </NButton>
        </div>
        <NInput
            v-model:value="licenseCode"
            class="mt-3"
            :placeholder="t('license.placeholder')"
            @keyup.enter="submitLicense"
        />
        <template #action>
          <NButton @click="quitApp">{{ t('license.quit') }}</NButton>
          <NButton type="primary" :loading="licenseLoading" @click="submitLicense">{{ t('license.submit') }}</NButton>
        </template>
      </NModal>
    </NaiveProvider>
    <NGlobalStyle/>
    <NModalProvider/>
  </NConfigProvider>
</template>

<script setup lang="ts">
import NaiveProvider from '@/components/NaiveProvider.vue'
import {darkTheme, lightTheme, zhCN, enUS} from 'naive-ui'
import {useIndexStore} from "@/stores"
import {computed, onMounted, ref} from "vue"
import {useEventStore} from "@/stores/event"
import type {appType} from "@/types/app"
import {useI18n} from 'vue-i18n'
import {ClipboardSetText, Quit, WindowSetTitle} from '../wailsjs/runtime'
import {CopyOutline} from '@vicons/ionicons5'
import appApi from '@/api/app'
import IssuerView from '@/views/issuer.vue'

const store = useIndexStore()
const eventStore = useEventStore()
const {locale, t} = useI18n()
const showLicense = ref(false)
const isIssuer = ref(!!(window as any).go?.core?.Issuer)
const licenseCode = ref("")
const licenseLoading = ref(false)
const machineCode = ref("")

const theme = computed(() => {
  if (store.globalConfig.Theme === "darkTheme") {
    document.documentElement.classList.add('dark');
    return darkTheme
  }
  document.documentElement.classList.remove('dark');
  return lightTheme
})

const uiLocale = computed(() => {
  locale.value = store.globalConfig.Locale
  if (store.globalConfig.Locale === "zh") {
    return zhCN
  }
  return enUS
})

const quitApp = () => {
  Quit()
}

const copyMachine = () => {
  if (!machineCode.value) {
    return
  }
  ClipboardSetText(machineCode.value).then(() => {
    window.$message?.success(t('common.copy_success'))
  })
}

const submitLicense = async () => {
  const code = licenseCode.value.trim()
  if (!code) {
    window.$message?.error(t('license.empty'))
    return
  }
  licenseLoading.value = true
  try {
    const res = await appApi.activateLicense(code)
    if (res.code !== 1) {
      window.$message?.error(res.message || t('license.empty'))
      return
    }
    showLicense.value = false
    const until = res.data?.expires ? t('license.until', {date: res.data.expires}) : t('license.ok')
    window.$message?.success(until)
  } finally {
    licenseLoading.value = false
  }
}

onMounted(async () => {
  if (isIssuer.value) {
    WindowSetTitle(t('issuer.title'))
    return
  }
  WindowSetTitle(t('app.name'))
  await store.init()
  try {
    const res = await appApi.licenseStatus()
    machineCode.value = res.data?.machine || ""
    showLicense.value = !(res.code === 1 && res.data?.ok)
  } catch {
    showLicense.value = true
  }

  eventStore.init()
  eventStore.addHandle({
    type: "message",
    event: (res: appType.Message) => {
      switch (res?.code) {
        case 0:
          window.$message?.error(res.message)
          break
        case 1:
          window.$message?.success(res.message)
          break
      }
    }
  })
})
</script>