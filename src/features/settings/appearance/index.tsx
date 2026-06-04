import { ContentSection } from '../components/content-section'
import { AppearanceForm } from './appearance-form'

export function SettingsAppearance() {
  return (
    <ContentSection
      title='外观'
      desc='自定义界面主题，以及当前浏览器本地保存的字体偏好。'
    >
      <AppearanceForm />
    </ContentSection>
  )
}
