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

import React, { useContext, useEffect, useState } from 'react';
import { Button, Typography, Input, ScrollList, ScrollItem, Row, Col } from '@douyinfe/semi-ui';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile.js';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { IconGithubLogo, IconCopy, IconPlay } from '@douyinfe/semi-icons';
import { IconAccessibility, IconBadgeStar, IconTag, IconColorPlatte } from '@douyinfe/semi-icons-lab';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import { Moonshot, OpenAI, XAI, Zhipu, Volcengine, Cohere, Claude, Gemini, Suno, Minimax, Wenxin, Spark, Qingyan, DeepSeek, Qwen, Midjourney, Grok, AzureAI, Hunyuan, Xinference } from '@lobehub/icons';
import './home.css';
import '../../index.css';

const { Text } = Typography;

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const serverAddress = statusState?.status?.server_address || window.location.origin;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const isChinese = i18n.language.startsWith('zh');

  // 打字机效果相关状态
  const [currentText, setCurrentText] = useState('');
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isDeleting, setIsDeleting] = useState(false);

  // 广告词数组
  const slogans = [
    t('一键接入，极致体验'),
    t('统一接口，无缝切换'),
    t('更优价格，更强兼容'),
    t('简单配置，即刻使用'),
    t('多模型聚合，一站服务')
  ];

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          const theme = localStorage.getItem('theme-mode') || 'light';
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: theme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent(t('加载首页内容失败...'));
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error(t('获取公告失败:'), error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  // 语言切换时重置打字机效果
  useEffect(() => {
    setCurrentText('');
    setCurrentIndex(0);
    setIsDeleting(false);
  }, [i18n.language]);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  // 打字机效果
  useEffect(() => {
    const currentSlogan = slogans[currentIndex];

    const typewriterTimer = setTimeout(() => {
      if (!isDeleting) {
        // 打字阶段
        if (currentText.length < currentSlogan.length) {
          setCurrentText(currentSlogan.slice(0, currentText.length + 1));
        } else {
          // 完成打字，等待一段时间后开始删除
          setTimeout(() => setIsDeleting(true), 2000);
        }
      } else {
        // 删除阶段
        if (currentText.length > 0) {
          setCurrentText(currentText.slice(0, -1));
        } else {
          // 删除完成，切换到下一个广告词
          setIsDeleting(false);
          setCurrentIndex((prev) => (prev + 1) % slogans.length);
        }
      }
    }, isDeleting ? 50 : 100); // 删除速度比打字速度快

    return () => clearTimeout(typewriterTimer);
  }, [currentText, currentIndex, isDeleting, slogans, t]);

  return (
    <div className="w-full overflow-x-hidden">
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className="w-full overflow-x-hidden">
          {/* Banner 部分 */}
          <div className="w-full border-b border-semi-color-border min-h-[500px] md:min-h-[600px] lg:min-h-[700px] relative overflow-x-hidden">
            {/* 背景模糊晕染球 */}
            <div className="gradient-circle"></div>
            <div className="flex items-center justify-center h-full px-4 py-20 md:py-24 lg:py-32 mt-10">
              {/* 居中内容区 */}
              <div className="flex flex-col items-center justify-center text-center max-w-4xl mx-auto">
                <div className="flex flex-col items-center justify-center mb-6 md:mb-8">
                  <h1 className={`text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-bold text-semi-color-text-0 leading-tight ${isChinese ? 'tracking-wide md:tracking-wider' : ''} relative z-10`}>
                    <span className="shine-text">MoleAPI</span>
                  </h1>
                  <h2 className="text-lg md:text-xl lg:text-2xl font-bold text-semi-color-text-1 mt-2 mb-4 relative z-10">
                    {t('统一的大模型网关接口')}
                  </h2>
                  <p className="text-sm md:text-base lg:text-lg text-semi-color-text-1 mt-4 md:mt-6 max-w-xl min-h-[1.5em] flex items-center justify-center">
                    <span className="typewriter-text">
                      {currentText}
                      <span className="typewriter-cursor">|</span>
                    </span>
                  </p>
                  {/* BASE URL 与端点选择 */}
                  <div className="flex flex-col md:flex-row items-center justify-center gap-4 w-full mt-4 md:mt-6 max-w-md">
                    <Input
                      readonly
                      value={serverAddress}
                      className="flex-1 !rounded-full"
                      size={isMobile ? 'default' : 'large'}
                      suffix={
                        <div className="flex items-center gap-2">
                          <ScrollList bodyHeight={32} style={{ border: 'unset', boxShadow: 'unset' }}>
                            <ScrollItem
                              mode="wheel"
                              cycled={true}
                              list={endpointItems}
                              selectedIndex={endpointIndex}
                              onSelect={({ index }) => setEndpointIndex(index)}
                            />
                          </ScrollList>
                          <Button
                            type="primary"
                            onClick={handleCopyBaseURL}
                            icon={<IconCopy />}
                            className="!rounded-full"
                          />
                        </div>
                      }
                    />
                  </div>
                </div>

                {/* 操作按钮 */}
                <div className="flex flex-row gap-4 justify-center items-center">
                  <Link to="/console">
                    <Button theme="solid" type="primary" size={isMobile ? "default" : "large"} className="!rounded-3xl px-8 py-2 relative" icon={<IconColorPlatte style={{ pointerEvents: 'none' }} />}>
                      {t('免费试用')}
                    </Button>
                  </Link>
                  {isDemoSiteMode && statusState?.status?.version ? (
                    <Button
                      size={isMobile ? "default" : "large"}
                      className="flex items-center !rounded-3xl px-6 py-2"
                      icon={<IconGithubLogo />}
                      onClick={() => window.open('https://github.com/QuantumNous/new-api', '_blank')}
                    >
                      {statusState.status.version}
                    </Button>
                  ) : (
                    <Link to="/pricing">
                      <Button
                        size={isMobile ? "default" : "large"}
                        className="flex items-center !rounded-3xl px-6 py-2 relative"
                        icon={<IconTag style={{ pointerEvents: 'none' }} />}
                      >
                        {t('了解价格')}
                      </Button>
                    </Link>
                  )}
                </div>

                {/* 框架兼容性图标 */}
                <div className="mt-12 md:mt-16 lg:mt-20 w-full">
                  <div className="flex items-center mb-6 md:mb-8 justify-center">
                    <Text type="tertiary" className="text-lg md:text-xl lg:text-2xl font-light">
                      {t('支持众多的大模型供应商')}
                    </Text>
                  </div>
                  <div className="flex flex-wrap items-center justify-center gap-3 sm:gap-4 md:gap-6 lg:gap-8 max-w-5xl mx-auto px-4">
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Moonshot size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <OpenAI size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <XAI size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Zhipu.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Volcengine.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Cohere.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Claude.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Gemini.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Suno size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Minimax.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Wenxin.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Spark.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Qingyan.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <DeepSeek.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Qwen.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Midjourney size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Grok size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <AzureAI.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Hunyuan.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Xinference.Color size={40} />
                    </div>
                    <div className="w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center">
                      <Typography.Text className="!text-lg sm:!text-xl md:!text-2xl lg:!text-3xl font-bold">30+</Typography.Text>
                    </div>
                  </div>
                </div>

                {/* 功能特性卡片 */}
                <div className="mt-16 md:mt-20 lg:mt-24 w-full max-w-6xl mx-auto px-4">
                  <Row gutter={[24, 24]}>
                    <Col xs={24} sm={8}>
                      <div className="feature-card text-center p-6 rounded-xl bg-semi-color-bg-2 shadow-lg hover:shadow-xl transition-all duration-300 hover:-translate-y-1">
                        <IconAccessibility size="extra-large" className="mb-4 text-semi-color-primary" />
                        <Typography.Title heading={4} className="mb-3 text-semi-color-text-0">{t('多平台支持')}</Typography.Title>
                        <Typography.Text className="text-semi-color-text-1">{t('支持OpenAI、Anthropic Claude、Google Gemini等多个 AI 平台')}</Typography.Text>
                      </div>
                    </Col>
                    <Col xs={24} sm={8}>
                      <div className="feature-card text-center p-6 rounded-xl bg-semi-color-bg-2 shadow-lg hover:shadow-xl transition-all duration-300 hover:-translate-y-1">
                        <IconBadgeStar size="extra-large" className="mb-4 text-semi-color-primary" />
                        <Typography.Title heading={4} className="mb-3 text-semi-color-text-0">{t('安全管理')}</Typography.Title>
                        <Typography.Text className="text-semi-color-text-1">{t('提供 API 密钥二次分发管理，支持 KEY 级额度控制，成本可控')}</Typography.Text>
                      </div>
                    </Col>
                    <Col xs={24} sm={8}>
                      <div className="feature-card text-center p-6 rounded-xl bg-semi-color-bg-2 shadow-lg hover:shadow-xl transition-all duration-300 hover:-translate-y-1">
                        <IconTag size="extra-large" className="mb-4 text-semi-color-primary" />
                        <Typography.Title heading={4} className="mb-3 text-semi-color-text-0">{t('一站式服务')}</Typography.Title>
                        <Typography.Text className="text-semi-color-text-1">{t('多个 AI 平台聚合，统一出口 API 端点，便捷的一站式 AI 服务体验')}</Typography.Text>
                      </div>
                    </Col>
                  </Row>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="overflow-x-hidden w-full">
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className="w-full h-screen border-none"
            />
          ) : (
            <div className="mt-[60px]" dangerouslySetInnerHTML={{ __html: homePageContent }} />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;

