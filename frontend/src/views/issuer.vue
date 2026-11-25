<template>
  <div class="h-full bg-slate-100 dark:bg-slate-950">
    <div class="flex items-center px-4 pt-3" style="--wails-draggable:drag">
      <div style="--wails-draggable:no-drag">
        <Screen/>
      </div>
      <div class="ml-3 text-sm font-semibold text-slate-700 dark:text-slate-200">{{ t('issuer.title') }}</div>
    </div>
    <div class="px-8 pb-8 pt-4" style="--wails-draggable:no-drag">
      <div class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
        <div class="flex items-center gap-4 bg-gradient-to-r from-blue-600 to-cyan-500 px-6 py-5 text-white">
          <img src="@/assets/image/logo.png" alt="" class="h-14 w-14 rounded-2xl bg-white/20 p-1"/>
          <div>
            <div class="text-lg font-semibold">{{ t('issuer.title') }}</div>
            <div class="mt-1 text-sm text-white/80">{{ t('issuer.subtitle') }}</div>
          </div>
        </div>
        <div class="px-6 py-5">
          <div class="text-sm text-slate-500">{{ t('license.machine') }}</div>
          <NInput
              v-model:value="machine"
              class="mt-2"
              size="large"
              :placeholder="t('issuer.machine_placeholder')"
          >
            <template #prefix>
              <NIcon size="18"><DesktopOutline/></NIcon>
            </template>
          </NInput>
          <div class="mt-4 text-sm text-slate-500">{{ t('issuer.days') }}</div>
          <NInputNumber v-model:value="days" class="mt-2 w-full" size="large" :min="0" :max="3650"/>
          <NButton class="mt-4" type="primary" size="large" block :loading="loading" @click="issue">
            <template #icon>
              <NIcon><KeyOutline/></NIcon>
            </template>
            {{ t('issuer.generate') }}
          </NButton>
          <div v-if="code" class="mt-5 rounded-xl bg-slate-50 px-4 py-3 dark:bg-slate-800">
            <div class="text-xs text-slate-500">{{ t('issuer.result') }}</div>
            <div class="mt-1 break-all font-mono text-sm">{{ code }}</div>
            <NButton class="mt-3" secondary type="primary" @click="copyCode">
              <template #icon>
                <NIcon><CopyOutline/></NIcon>
              </template>
              {{ t('issuer.copy') }}
            </NButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {ref} from "vue"
import {useI18n} from "vue-i18n"
import {CopyOutline, DesktopOutline, KeyOutline} from "@vicons/ionicons5"
import {ClipboardSetText} from "../../wailsjs/runtime"
import Screen from "@/components/Screen.vue"

const {t} = useI18n()
const machine = ref("")
const days = ref(0)
const code = ref("")
const loading = ref(false)

const issue = async () => {
  const id = machine.value.trim()
  if (!id) {
    window.$message?.error(t("issuer.machine_empty"))
    return
  }
  loading.value = true
  try {
    const res = await (window as any).go.core.Issuer.Issue(id, days.value || 0)
    if (res.code !== 1) {
      window.$message?.error(res.message)
      return
    }
    code.value = res.data.code
  } finally {
    loading.value = false
  }
}

const copyCode = () => {
  if (!code.value) {
    return
  }
  ClipboardSetText(code.value).then(() => {
    window.$message?.success(t("common.copy_success"))
  })
}
</script>
