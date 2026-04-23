/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import HeaderBar from './headerbar';
import { Layout } from '@douyinfe/semi-ui';
import SiderBar from './SiderBar';
import App from '../../App';
import FooterBar from './Footer';
import { ToastContainer } from 'react-toastify';
import ErrorBoundary from '../common/ErrorBoundary';
import React, { useContext, useEffect, useState } from 'react';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import { useTranslation } from 'react-i18next';
import {
  API,
  getDisplaySiteName,
  getLogo,
  getSystemName,
  showError,
  setStatusData,
} from '../../helpers';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useLocation } from 'react-router-dom';
import { normalizeLanguage } from '../../i18n/language';
const { Sider, Content, Header } = Layout;

const PageLayout = () => {
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const isMobile = useIsMobile();
  const [collapsed, , setCollapsed] = useSidebarCollapsed();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { i18n } = useTranslation();
  const location = useLocation();

  const cardProPages = [
    '/console/channel',
    '/console/log',
    '/console/redemption',
    '/console/user',
    '/console/token',
    '/console/midjourney',
    '/console/task',
    '/console/models',
    '/pricing',
  ];

  const shouldHideFooter = cardProPages.includes(location.pathname);

  const shouldInnerPadding =
    location.pathname.includes('/console') &&
    !location.pathname.startsWith('/console/chat') &&
    location.pathname !== '/console/playground';

  const isConsoleRoute = location.pathname.startsWith('/console');
  const showSider = isConsoleRoute && (!isMobile || drawerOpen);

  useEffect(() => {
    if (isMobile && drawerOpen && collapsed) {
      setCollapsed(false);
    }
  }, [isMobile, drawerOpen, collapsed, setCollapsed]);

  const loadUser = () => {
    let user = localStorage.getItem('user');
    if (user) {
      let data = JSON.parse(user);
      userDispatch({ type: 'login', payload: data });
    }
  };

  const loadStatus = async () => {
    try {
      const res = await API.get('/api/status');
      const { success, data } = res.data;
      if (success) {
        statusDispatch({ type: 'set', payload: data });
        setStatusData(data);
      } else {
        showError('Unable to connect to server');
      }
    } catch (error) {
      showError('Failed to load status');
    }
  };

  useEffect(() => {
    loadUser();
    loadStatus().catch(console.error);
    let logo = getLogo();
    if (logo) {
      let linkElement = document.querySelector("link[rel~='icon']");
      if (linkElement) {
        linkElement.href = logo;
      }
    }
  }, []);

  useEffect(() => {
    const systemName = getDisplaySiteName(
      statusState?.status?.system_name || getSystemName(),
    );
    const isChinese = i18n.language.startsWith('zh');
    const routeMetaMap = {
      '/': {
        title: isChinese
          ? `${systemName} 控制台 | 高速稳定的大模型服务入口`
          : `${systemName} Console | Fast and Stable AI Model Access`,
        description: isChinese
          ? `${systemName} 是 MoleAPI 的控制台入口，提供高速、稳定的大模型服务接入体验，帮助团队统一连接 OpenAI、Anthropic、Google 等真实官方模型供应商。`
          : `${systemName} is the MoleAPI console for fast, stable access to AI services from real official providers such as OpenAI, Anthropic, and Google.`,
      },
      '/pricing': {
        title: isChinese
          ? `${systemName} 价格与充值 | MoleAPI 控制台`
          : `${systemName} Pricing and Top-up | MoleAPI Console`,
        description: isChinese
          ? '查看 MoleAPI 控制台的充值与价格信息，支持按实付金额开票，适合团队和开发者使用。'
          : 'Review MoleAPI console pricing and top-up options with invoice-ready payments for teams and developers.',
      },
      '/about': {
        title: isChinese
          ? `关于 ${systemName} | MoleAPI 控制台`
          : `About ${systemName} | MoleAPI Console`,
        description: isChinese
          ? `${systemName} 聚焦统一 API 网关、额度管理、账单与官方模型供应商接入能力。`
          : `${systemName} focuses on unified API gateway access, quota management, billing, and official model provider connectivity.`,
      },
      '/login': {
        title: isChinese
          ? `${systemName} 登录 | MoleAPI 控制台`
          : `${systemName} Login | MoleAPI Console`,
        description: isChinese
          ? `登录 ${systemName}，进入 MoleAPI 控制台管理模型调用、额度、账单与团队配置。`
          : `Sign in to ${systemName} and manage model access, quotas, billing, and team settings in the MoleAPI console.`,
      },
      '/register': {
        title: isChinese
          ? `${systemName} 注册 | MoleAPI 控制台`
          : `${systemName} Register | MoleAPI Console`,
        description: isChinese
          ? `注册 ${systemName}，体验高速、稳定的大模型服务接入与统一 API 管理。`
          : `Create an account for ${systemName} and experience fast, stable AI model access with unified API management.`,
      },
    };
    const fallbackMeta = {
      title: isChinese
        ? `${systemName} | MoleAPI 大模型控制台`
        : `${systemName} | MoleAPI AI Model Console`,
      description: isChinese
        ? `${systemName} 提供高速、稳定的大模型服务接入体验，并连接真实官方模型供应商。`
        : `${systemName} provides fast, stable access to AI models from real official providers.`,
    };
    const currentMeta = routeMetaMap[location.pathname] || fallbackMeta;
    const canonicalUrl = `${window.location.origin}${location.pathname}`;

    document.title = currentMeta.title;
    document.documentElement.lang = i18n.language.startsWith('zh')
      ? 'zh-CN'
      : i18n.language;

    const setMetaContent = (selector, content) => {
      const element = document.querySelector(selector);
      if (element && content) {
        element.setAttribute('content', content);
      }
    };

    setMetaContent('meta[name="description"]', currentMeta.description);
    setMetaContent('meta[property="og:title"]', currentMeta.title);
    setMetaContent('meta[property="og:description"]', currentMeta.description);
    setMetaContent('meta[property="og:url"]', canonicalUrl);
    setMetaContent('meta[name="twitter:title"]', currentMeta.title);
    setMetaContent(
      'meta[name="twitter:description"]',
      currentMeta.description,
    );

    const canonicalLink = document.querySelector('link[rel="canonical"]');
    if (canonicalLink) {
      canonicalLink.setAttribute('href', canonicalUrl);
    }
  }, [i18n.language, location.pathname, statusState?.status?.system_name]);

  useEffect(() => {
    let preferredLang;

    if (userState?.user?.setting) {
      try {
        const settings = JSON.parse(userState.user.setting);
        preferredLang = normalizeLanguage(settings.language);
      } catch (e) {
        // Ignore parse errors
      }
    }

    if (!preferredLang) {
      const savedLang = localStorage.getItem('i18nextLng');
      if (savedLang) {
        preferredLang = normalizeLanguage(savedLang);
      }
    }

    if (preferredLang) {
      localStorage.setItem('i18nextLng', preferredLang);
      if (preferredLang !== i18n.language) {
        i18n.changeLanguage(preferredLang);
      }
    }
  }, [i18n, userState?.user?.setting]);

  return (
    <Layout
      className='app-layout'
      style={{
        display: 'flex',
        flexDirection: 'column',
        overflow: isMobile ? 'visible' : 'hidden',
      }}
    >
      <Header
        style={{
          padding: 0,
          height: 'auto',
          lineHeight: 'normal',
          position: 'fixed',
          width: '100%',
          top: 0,
          zIndex: 100,
        }}
      >
        <HeaderBar
          onMobileMenuToggle={() => setDrawerOpen((prev) => !prev)}
          drawerOpen={drawerOpen}
        />
      </Header>
      <Layout
        style={{
          overflow: isMobile ? 'visible' : 'auto',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {showSider && (
          <Sider
            className='app-sider'
            style={{
              position: 'fixed',
              left: 0,
              top: '64px',
              zIndex: 99,
              border: 'none',
              paddingRight: '0',
              width: 'var(--sidebar-current-width)',
            }}
          >
            <SiderBar
              onNavigate={() => {
                if (isMobile) setDrawerOpen(false);
              }}
            />
          </Sider>
        )}
        <Layout
          style={{
            marginLeft: isMobile
              ? '0'
              : showSider
                ? 'var(--sidebar-current-width)'
                : '0',
            flex: '1 1 auto',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Content
            style={{
              flex: '1 0 auto',
              overflowY: isMobile ? 'visible' : 'hidden',
              WebkitOverflowScrolling: 'touch',
              padding: shouldInnerPadding ? (isMobile ? '5px' : '24px') : '0',
              position: 'relative',
            }}
          >
            <ErrorBoundary>
              <App />
            </ErrorBoundary>
          </Content>
          {!shouldHideFooter && (
            <Layout.Footer
              style={{
                flex: '0 0 auto',
                width: '100%',
              }}
            >
              <FooterBar />
            </Layout.Footer>
          )}
        </Layout>
      </Layout>
      <ToastContainer />
    </Layout>
  );
};

export default PageLayout;
