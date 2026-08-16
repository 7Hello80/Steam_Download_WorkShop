import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

// Font Awesome
import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faDownload,
  faFolderOpen,
  faUser,
  faHeart,
  faCamera,
  faPenToSquare,
  faArrowRight,
  faRightFromBracket,
  faClose,
  faCircle,
  faCircleInfo,
  faCircleCheck,
  faCircleXmark,
  faTriangleExclamation,
  faSpinner,
  faGamepad,
  faMugHot,
  faGear,
  faShieldHalved,
  faLock,
  faEnvelope,
  faEye,
  faEyeSlash,
  faChevronRight,
  faCloudArrowDown,
  faFileZipper,
  faTrash,
  faClock,
  faCheck,
  faXmark,
  faExclamationTriangle,
  faExchangeAlt,
  faFilm,
  faVideo,
  faImage,
  faWeightHanging,
  faCog,
  faBullhorn,
  faPen,
  faAdd,
  faWandMagicSparkles,
  faCloudUploadAlt,
  faFileVideo,
  faCircleQuestion,
  faInfoCircle,
  faPlay,
  faBook,
  faTh,
  faMoneyBillWave,
  faCircleDollarToSlot,
  faCommentDots,
} from '@fortawesome/free-solid-svg-icons'
import { faGithub, faSteam, faWeixin } from '@fortawesome/free-brands-svg-icons'

import App from './App.vue'
import router from './router'
import './styles/variables.css'
import './styles/global.css'

// Add icons to library
library.add(
  faDownload,
  faFolderOpen,
  faUser,
  faHeart,
  faCamera,
  faPenToSquare,
  faArrowRight,
  faRightFromBracket,
  faClose,
  faCircle,
  faCircleInfo,
  faCircleCheck,
  faCircleXmark,
  faTriangleExclamation,
  faSpinner,
  faGamepad,
  faMugHot,
  faGear,
  faShieldHalved,
  faLock,
  faEnvelope,
  faEye,
  faEyeSlash,
  faChevronRight,
  faCloudArrowDown,
  faFileZipper,
  faTrash,
  faClock,
  faCheck,
  faXmark,
  faExclamationTriangle,
  faExchangeAlt,
  faFilm,
  faVideo,
  faImage,
  faWeightHanging,
  faCog,
  faBullhorn,
  faPen,
  faAdd,
  faWandMagicSparkles,
  faCloudUploadAlt,
  faFileVideo,
  faCircleQuestion,
  faInfoCircle,
  faPlay,
  faBook,
  faTh,
  faGithub,
  faSteam,
  faWeixin,
  faMoneyBillWave,
  faCircleDollarToSlot,
  faCommentDots,
)

const app = createApp(App)

// Register Font Awesome component globally
app.component('font-awesome-icon', FontAwesomeIcon)

// Register all Element Plus icons (keep for backward compat)
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
for (const [key, component] of Object.entries(ElementPlusIconsVue as Record<string, unknown>)) {
  app.component(key, component)
}

app.use(createPinia())
app.use(router)
app.use(ElementPlus, { size: 'default' })

app.mount('#app')
