import {createApp} from 'vue'
import App from './App.vue'
import './style.css';
import PrimeVue from 'primevue/config';
import 'primevue/resources/themes/md-light-indigo/theme.css';  // Material Design Theme
import 'primevue/resources/primevue.min.css';
import 'primeicons/primeicons.css';
import 'primeflex/primeflex.css';

import Button from 'primevue/button';
import Dropdown from 'primevue/dropdown';
import Toast from 'primevue/toast';
import ToastService from 'primevue/toastservice';
import Dialog from 'primevue/dialog';

const app = createApp(App);
app.use(PrimeVue);
app.use(ToastService);
app.component('Button', Button);
app.component('Dropdown', Dropdown);
app.component('Toast', Toast);
app.component('Dialog', Dialog);

app.mount('#app');
