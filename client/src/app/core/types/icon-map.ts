/**
 * Icon mapping based on the icon map from views/helpers/icons.go
 * Maps icon names to ng-icons icon identifiers
 *
 * Icon prefixes:
 * - fa6-solid = Font Awesome 6 Solid (@ng-icons/font-awesome/solid)
 * - fa6-regular = Font Awesome 6 Regular (@ng-icons/font-awesome/regular)
 * - material-symbols = Material Symbols (@ng-icons/material-symbols/outline)
 * - ion = Ionicons (@ng-icons/ionicons)
 * - mdi = Material Design Icons from Pictogrammers (@ng-icons/simple-icons - using similar alternatives)
 * - octicon = Octicons (@ng-icons/octicons)
 * - map = Custom map icons (fallback to similar icons)
 * - hugeicons = Huge Icons (fallback to similar icons)
 */

export const ICON_MAP: Record<string, string> = {
  // Category Icons
  source: 'faSolidBookmark',
  distance: 'faSolidRoad',
  speed: 'faSolidGauge',
  'max-speed': 'faSolidGaugeHigh',
  tempo: 'faSolidStopwatch',
  duration: 'faClock',
  elevation: 'faSolidMountain',
  location: 'faSolidMapLocationDot',
  repetitions: 'faSolidCalculator',
  weight: 'faSolidWeightHanging',
  'heart-rate': 'faSolidHeartPulse',
  cadence: 'faSolidStopwatch',
  temperature: 'faSolidTemperatureHigh',
  heading: 'faSolidCompass',
  accuracy: 'faSolidCrosshairs',
  date: 'faCalendar',
  name: 'faSolidTag',
  timezone: 'faSolidGlobe', // Using FA icon as matSymbolsMap not available
  pause: 'faHourglass',
  calories: 'faSolidFire',
  steps: 'ionFootsteps',
  scale: 'ionScale',
  height: 'faSolidRulerVertical', // Using FA icon as mdi-human-male-height not available
  trophy: 'faSolidTyrophy',

  // Misc Icons
  welcome: 'faSolidChevronRight',
  circular: 'faSolidCircleNotch',
  bidirectional: 'faSolidArrowRightArrowLeft',
  units: 'faSolidRuler',
  file: 'faSolidFile',
  best: 'faSolidArrowUpLong',
  worst: 'faSolidArrowDownLong',
  up: 'faSolidChevronUp',
  down: 'faSolidChevronDown',
  metrics: 'faRectangleList',
  translate: 'faSolidLanguage',
  expand: 'faSolidArrowsLeftRight',
  share: 'faSolidShareFromSquare',
  'generate-share': 'faSolidRetweet',
  logout: 'faSolidRightFromBracket',

  // Sport Icons
  cycling: 'faSolidBicycle',
  'e-cycling': 'faSolidBicycle', // Using bicycle as electric-bike not available
  running: 'faSolidPersonRunning',
  walking: 'faSolidPersonWalking',
  swimming: 'faSolidPersonSwimming',
  'inline-skating': 'faSolidPersonSkating', // Using closest FA alternative
  skiing: 'faSolidPersonSkiing',
  snowboarding: 'faSolidPersonSnowboarding',
  golfing: 'faSolidGolfBallTee',
  kayaking: 'faSolidSailboat',
  hiking: 'faSolidPersonHiking',
  'horse-riding': 'faSolidHorse', // Using FA icon as mdi-horse-human not available
  'push-ups': 'faSolidDumbbell', // Using dumbbell as hugeicons-push-up-bar not available
  'weight-lifting': 'faSolidDumbbell',
  rowing: 'faSolidWater', // Using water as matSymbolsRowing not available
  other: 'faSolidQuestion',

  // Page Icons
  dashboard: 'faSolidChartLine',
  statistics: 'faSolidChartSimple',
  admin: 'faSolidGear',
  actions: 'faSolidGear',
  'user-profile': 'faCircleUser',
  'user-add': 'faSolidUserPlus',
  workout: 'faSolidDumbbell',
  equipment: 'faSolidBicycle',
  'route-segment': 'faSolidRoute',
  add: 'faSolidCirclePlus',
  'workout-add': 'faSolidCirclePlus',
  'equipment-add': 'faSolidCirclePlus',
  'route-segment-add': 'faSolidCirclePlus',
  heatmap: 'faSolidFire',
  changelog: 'faSolidClipboardCheck', // Using FA icon as mdi-clipboard-check not available

  // Utility Icons
  close: 'faSolidXmark',
  edit: 'faSolidPenToSquare',
  back: 'faSolidArrowLeft',
  view: 'faSolidEye',
  'auto-update': 'faSolidArrowsRotate',
  refresh: 'faSolidArrowsRotate',
  delete: 'faSolidTrash',
  note: 'faSolidQuoteLeft',
  users: 'faSolidUsers',
  'user-signin': 'faSolidRightToBracket',
  'user-signout': 'faSolidRightFromBracket',
  'user-register': 'faSolidUserPlus',
  user: 'faSolidUser',
  person: 'faSolidUser',
  show: 'faSolidEye',
  hide: 'faSolidEyeSlash',
  copy: 'faSolidClipboard',
  download: 'faSolidDownload',
  attention: 'faSolidCircleExclamation',
  check: 'faSolidSquareCheck',
  totals: 'faSolidCalculator',
  missing: 'faSolidBan', // Using ban as matSymbolsBlock not available
  locked: 'faSolidUserLock',
  unlocked: 'faSolidLockOpen',
  settings: 'faSolidGear',
  info: 'faSolidCircleInfo',
  visibility: 'faSolidEye',
  'visibility-off': 'faSolidEyeSlash',
  'content-copy': 'faSolidClipboard',
  straighten: 'faSolidRuler',
  bolt: 'faSolidBolt',

  // Brand Icons
  github: 'octMarkGithub',

  // Menu toggle icon
  menu: 'faSolidBars',
};

/**
 * Get the ng-icons icon name for a given icon key
 * @param key The icon key from the icon map
 * @returns The ng-icons icon name, or a default question icon if not found
 */
export function getIcon(key: string): string {
  return ICON_MAP[key] || 'faSolidQuestion';
}
