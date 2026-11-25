<template>
  <NModal
      :show="showModal"
      :on-update:show="changeShow"
      style="--wails-draggable:no-drag"
      preset="card"
      class="w-[640px]"
      :title="t('footer.title')"
  >
    <div class="rounded p-5">
      <div class="flex flex-col">
        <div class="flex flex-row">
          <div>
            <div class="text-2xl font-bold">{{ t('footer.headline') }}</div>
            <div class="text-slate-400 w-96 pt-3 leading-7">
              {{ t('footer.description') }}
            </div>
          </div>
          <div class="pl-8">
            <img src="@/assets/image/logo.png" alt="Logo" class="h-28 w-28"/>
          </div>
        </div>
      </div>
      <div class="flex w-full text-sm justify-between pt-8 text-slate-400">
        <div>{{ t('footer.author_label') }} {{ t('footer.name') }}</div>
        <button @click="toWebsite(certUrl)">{{ t('footer.cert_download') }}</button>
      </div>
    </div>
  </NModal>
</template>

<script lang="ts" setup>
import {useIndexStore} from "@/stores"
import {BrowserOpenURL} from "../../wailsjs/runtime"
import {computed} from "vue"
import {useI18n} from 'vue-i18n'

const {t} = useI18n()
const store = useIndexStore()
const props = defineProps(["showModal"])
const emits = defineEmits(["update:showModal"])
const certUrl = computed(()=>{
  return store.baseUrl + "/api/cert"
})
const changeShow = (value: boolean) => {
  emits('update:showModal', value)
}

const toWebsite = (url: string) => {
  BrowserOpenURL(url)
}
</script>