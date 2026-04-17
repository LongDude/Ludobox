import { ref } from 'vue'

export type Locale = 'en' | 'ru'

const STORAGE_KEY = 'app.locale'

function loadLocale(): Locale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'en' || saved === 'ru') return saved
  } catch {}
  return 'en'
}

export const locale = ref<Locale>(loadLocale())

export function setLocale(next: Locale) {
  locale.value = next
  try {
    localStorage.setItem(STORAGE_KEY, next)
  } catch {}
}

const messages: Record<Locale, Record<string, string>> = {
  en: {
    'nav.adminPanel': 'Admin Panel',
    'nav.settings': 'Settings',

    'settings.title': 'Settings',
    'settings.appearance': 'Appearance',
    'settings.dark': 'Dark theme',
    'settings.light': 'Light theme',
    'settings.sidebar': 'Sidebar',
    'settings.hideLeft': 'Hide left panel',
    'settings.language': 'Language',
    'settings.langEnglish': 'English',
    'settings.langRussian': 'Russian',

    'common.add': 'Add',
    'common.remove': 'Remove',
    'common.submit': 'Submit',
    'common.submitting': 'Submitting…',
    'common.refresh': 'Refresh',
    'common.loading': 'Loading…',
    'common.edit': 'Edit',
    'common.cancel': 'Cancel',
    'common.save': 'Save',
    'common.apply': 'Apply',
    'common.delete': 'Delete',
    'common.reset': 'Reset',

    'admin.title': 'Admin Panel',
    'admin.toModerator': 'Moderator Panel',
    'admin.users': 'Users',
    'admin.noUsers': 'No users',
    'admin.errFetch': 'Failed to fetch users',
    'admin.columns.email': 'Email',
    'admin.columns.first': 'First name',
    'admin.columns.last': 'Last name',
    'admin.columns.locale': 'Locale',
    'admin.columns.confirmed': 'Confirmed',
    'admin.columns.photo': 'Photo',
    'admin.columns.roles': 'Roles',
    'admin.columns.password': 'Password',
    'admin.filters.search': 'Search',
    'admin.filters.role': 'Role',
    'admin.filters.locale': 'Locale',
    'admin.filters.confirmed': 'Email confirmed',
    'admin.filters.limit': 'Page size',
    'admin.filters.apply': 'Apply filters',
    'admin.filters.reset': 'Reset',
    'admin.pager.prev': 'Prev',
    'admin.pager.next': 'Next',
    'admin.pager.pageOf': 'Page {page} of {pages}',
    'admin.pager.total': 'Total: {total}',

    'auth.login': 'Log in',
    'auth.signup': 'Sign up',
    'auth.email': 'Email',
    'auth.password': 'Password',
    'auth.confirmPassword': 'Confirm Password',
    'auth.firstname': 'Firstname',
    'auth.lastname': 'Lastname',
    'auth.forgot': 'Forgot Password?',
    'auth.resetEmailRequired': 'Please enter your email first.',
    'auth.resetSuccess': 'If the email exists, we sent password reset instructions.',
    'auth.resetFailed': 'Could not send reset instructions. Try again later.',
    'auth.resetTitle': 'Reset password',
    'auth.resetDescription':
      'Enter your email and we will send a reset link if the account exists.',
    'auth.resetSubmit': 'Send reset link',
    'auth.resetCancel': 'Cancel',
    'auth.resetClose': 'Close',
    'auth.continueGoogle': 'Continue with Google',
    'auth.continueYandex': 'Continue with Yandex',
    'auth.noAccount': "Don't have an account?",
    'auth.haveAccount': 'Already have an account?',
    'auth.createOne': 'Create one',
    'auth.signIn': 'Sign in',

    'notFound.title': '404 - Page Not Found',
    'notFound.desc': 'The page you are looking for does not exist.',
    'notFound.home': 'Go back to Home',

    'footer.copy': '© 2025 LiveisFPV Dev. All rights reserved.',

    'search.title': 'Search View',
    'search.placeholder': 'This is a placeholder page.',

    'profile.title': 'Your profile',
    'profile.emailConfirmed': 'Email confirmed',
    'profile.locale': 'Locale',
    'profile.roles': 'Roles',
    'profile.form.firstName': 'First name',
    'profile.form.lastName': 'Last name',
    'profile.form.locale': 'Locale (e.g. en, ru)',
    'profile.form.newPassword': 'New password',
    'profile.form.keepBlank': 'Leave blank to keep',
    'profile.btn.cancel': 'Cancel',
    'profile.btn.save': 'Save changes',
    'profile.saving': 'Saving...',
    'profile.btn.edit': 'Edit profile',
    'profile.btn.logout': 'Log out',
    'profile.msg.nothing': 'Nothing to update',
    'profile.msg.updated': 'Profile updated',
    'profile.msg.failed': 'Failed to update profile',
    'common.yes': 'Yes',
    'common.no': 'No',
  },
  ru: {
    'nav.adminPanel': 'Админ-панель',
    'nav.settings': 'Настройки',

    'settings.title': 'Настройки',
    'settings.appearance': 'Оформление',
    'settings.dark': 'Тёмная тема',
    'settings.light': 'Светлая тема',
    'settings.sidebar': 'Боковая панель',
    'settings.hideLeft': 'Скрывать левую панель',
    'settings.language': 'Язык',
    'settings.langEnglish': 'Английский',
    'settings.langRussian': 'Русский',

    'common.add': 'Добавить',
    'common.remove': 'Удалить',
    'common.submit': 'Отправить',
    'common.submitting': 'Отправка…',
    'common.refresh': 'Обновить',
    'common.loading': 'Загрузка…',
    'common.edit': 'Редактировать',
    'common.cancel': 'Отмена',
    'common.save': 'Сохранить',
    'common.apply': 'Применить',
    'common.delete': 'Удалить',
    'common.reset': 'Сбросить',

    'admin.title': 'Панель администратора',
    'admin.toModerator': 'Панель модератора',
    'admin.users': 'Пользователи',
    'admin.noUsers': 'Нет пользователей',
    'admin.errFetch': 'Не удалось получить список пользователей',
    'admin.columns.email': 'Email',
    'admin.columns.first': 'Имя',
    'admin.columns.last': 'Фамилия',
    'admin.columns.locale': 'Язык',
    'admin.columns.confirmed': 'Подтв. email',
    'admin.columns.photo': 'Фото',
    'admin.columns.roles': 'Роли',
    'admin.columns.password': 'Пароль',
    'admin.filters.search': 'Поиск',
    'admin.filters.role': 'Роль',
    'admin.filters.locale': 'Язык',
    'admin.filters.confirmed': 'Email подтверждён',
    'admin.filters.limit': 'Размер страницы',
    'admin.filters.apply': 'Применить фильтры',
    'admin.filters.reset': 'Сбросить',
    'admin.pager.prev': 'Назад',
    'admin.pager.next': 'Вперёд',
    'admin.pager.pageOf': 'Стр. {page} из {pages}',
    'admin.pager.total': 'Всего: {total}',

    'auth.login': 'Войти',
    'auth.signup': 'Зарегистрироваться',
    'auth.email': 'Email',
    'auth.password': 'Пароль',
    'auth.confirmPassword': 'Подтвердите пароль',
    'auth.firstname': 'Имя',
    'auth.lastname': 'Фамилия',
    'auth.forgot': 'Забыли пароль?',
    'auth.resetEmailRequired': 'Введите email, чтобы сбросить пароль.',
    'auth.resetSuccess': 'Если такой email существует, мы отправили инструкции по восстановлению.',
    'auth.resetFailed': 'Не удалось отправить письмо. Попробуйте позже.',
    'auth.resetTitle': 'Восстановление пароля',
    'auth.resetDescription':
      'Введите email, и мы отправим ссылку для восстановления, если аккаунт существует.',
    'auth.resetSubmit': 'Отправить ссылку',
    'auth.resetCancel': 'Отмена',
    'auth.resetClose': 'Закрыть',
    'auth.continueGoogle': 'Войти через Google',
    'auth.continueYandex': 'Войти через Яндекс',
    'auth.noAccount': 'Нет аккаунта?',
    'auth.haveAccount': 'Уже есть аккаунт?',
    'auth.createOne': 'Создать',
    'auth.signIn': 'Войти',

    'notFound.title': '404 - Страница не найдена',
    'notFound.desc': 'Страница, которую вы ищете, не существует.',
    'notFound.home': 'Вернуться на главную',

    'footer.copy': '© 2025 LiveisFPV Dev. Все права защищены.',

    'search.title': 'Поиск',
    'search.placeholder': 'Временная страница-заглушка.',

    'profile.title': 'Ваш профиль',
    'profile.emailConfirmed': 'Email подтверждён',
    'profile.locale': 'Язык',
    'profile.roles': 'Роли',
    'profile.form.firstName': 'Имя',
    'profile.form.lastName': 'Фамилия',
    'profile.form.locale': 'Язык (например, en, ru)',
    'profile.form.newPassword': 'Новый пароль',
    'profile.form.keepBlank': 'Оставьте пустым, чтобы не менять',
    'profile.btn.cancel': 'Отмена',
    'profile.btn.save': 'Сохранить изменения',
    'profile.saving': 'Сохранение…',
    'profile.btn.edit': 'Редактировать профиль',
    'profile.btn.logout': 'Выйти',
    'profile.msg.nothing': 'Нечего обновлять',
    'profile.msg.updated': 'Профиль обновлён',
    'profile.msg.failed': 'Не удалось обновить профиль',
    'common.yes': 'Да',
    'common.no': 'Нет',
  },
}

export function t(key: string): string {
  const l = locale.value
  return (messages[l] && messages[l][key]) ?? messages.en[key] ?? key
}

export function useI18n() {
  return { locale, setLocale, t, messages }
}
