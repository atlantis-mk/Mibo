import { ContentSection } from '../components/content-section'
import { AccountForm } from './account-form'

export function SettingsAccount() {
  return (
    <ContentSection
      title='账户'
      desc='更新界面语言偏好，以及默认音频和字幕语言选择。'
    >
      <AccountForm />
    </ContentSection>
  )
}
