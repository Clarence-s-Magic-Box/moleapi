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

import React, { useEffect, useState, useContext, useRef } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  renderQuota,
  renderQuotaWithAmount,
  copy,
  getQuotaPerUnit,
} from '../../helpers';
import { Modal, Toast } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { QRCodeSVG } from 'qrcode.react';

import RechargeCard from './RechargeCard';
import InvitationCard from './InvitationCard';
import TransferModal from './modals/TransferModal';
import PaymentConfirmModal from './modals/PaymentConfirmModal';
import TopupHistoryModal from './modals/TopupHistoryModal';

const TopUp = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const LEGACY_PRESET_AMOUNTS = [1, 3, 7, 15, 35, 70, 140, 280];

  const [redemptionCode, setRedemptionCode] = useState('');
  const [amount, setAmount] = useState(0.0);
  const [minTopUp, setMinTopUp] = useState(statusState?.status?.min_topup || 1);
  const [topUpCount, setTopUpCount] = useState(
    statusState?.status?.min_topup || 1,
  );
  const [topUpLink, setTopUpLink] = useState(
    statusState?.status?.top_up_link || '',
  );
  const [enableOnlineTopUp, setEnableOnlineTopUp] = useState(
    statusState?.status?.enable_online_topup || false,
  );
  const [priceRatio, setPriceRatio] = useState(statusState?.status?.price || 1);

  const [enableStripeTopUp, setEnableStripeTopUp] = useState(
    statusState?.status?.enable_stripe_topup || false,
  );
  const [statusLoading, setStatusLoading] = useState(true);

  // Creem 相关状态
  const [creemProducts, setCreemProducts] = useState([]);
  const [enableCreemTopUp, setEnableCreemTopUp] = useState(false);
  const [creemOpen, setCreemOpen] = useState(false);
  const [selectedCreemProduct, setSelectedCreemProduct] = useState(null);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [payWay, setPayWay] = useState('');
  const [amountLoading, setAmountLoading] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [payMethods, setPayMethods] = useState([]);

  // LanTu (WeChat) QR modal + polling (desktop native pay)
  const [lantuPayModalOpen, setLantuPayModalOpen] = useState(false);
  const [lantuPayUrl, setLantuPayUrl] = useState('');
  const [lantuPayUrlKind, setLantuPayUrlKind] = useState('');
  const [lantuTradeNo, setLantuTradeNo] = useState('');
  const [lantuCountdown, setLantuCountdown] = useState(0);
  const [lantuPolling, setLantuPolling] = useState(false);

  const affFetchedRef = useRef(false);

  // 邀请相关状态
  const [affLink, setAffLink] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(0);

  // 账单Modal状态
  const [openHistory, setOpenHistory] = useState(false);

  // 订阅相关
  const [subscriptionPlans, setSubscriptionPlans] = useState([]);
  const [subscriptionLoading, setSubscriptionLoading] = useState(true);
  const [billingPreference, setBillingPreference] =
    useState('subscription_first');
  const [activeSubscriptions, setActiveSubscriptions] = useState([]);
  const [allSubscriptions, setAllSubscriptions] = useState([]);

  // 预设充值额度选项
  const [presetAmounts, setPresetAmounts] = useState([]);
  const [selectedPreset, setSelectedPreset] = useState(null);
  const [presetPayAmountMap, setPresetPayAmountMap] = useState({});

  // 充值配置信息
  const [topupInfo, setTopupInfo] = useState({
    amount_options: [],
    bonus: {},
    discount: {},
  });

  // 当前用户的充值分组倍率（由 /api/user/topup/info 下发；普通用户无权读取 /api/option）
  const [topupGroupRatio, setTopupGroupRatio] = useState(1);

  const getBonusRateForAmount = (bonusMap, amountValue) => {
    if (!bonusMap) return 0;
    // 后端可能返回 string（历史/配置兼容），这里做一次兜底解析
    if (typeof bonusMap === 'string') {
      try {
        bonusMap = JSON.parse(bonusMap);
      } catch (e) {
        return 0;
      }
    }
    const amount = Number(amountValue);
    if (!Number.isFinite(amount)) return 0;

    const direct = bonusMap[amount];
    if (typeof direct === 'number' && direct > 0) return direct;

    // Tier matching: take the highest key <= amount.
    const keys = Object.keys(bonusMap)
      .map((k) => Number(k))
      .filter((n) => Number.isFinite(n))
      .sort((a, b) => a - b);
    let best = 0;
    for (const k of keys) {
      if (k <= amount) {
        const v = Number(bonusMap[k]);
        if (Number.isFinite(v) && v > 0) best = v;
      }
    }
    return best;
  };

  const parseBonusMap = (rawBonus) => {
    if (!rawBonus) return {};
    if (typeof rawBonus === 'string') {
      try {
        const parsed = JSON.parse(rawBonus);
        return parsed && typeof parsed === 'object' ? parsed : {};
      } catch (e) {
        return {};
      }
    }
    return typeof rawBonus === 'object' ? rawBonus : {};
  };

  const parseDiscountMap = (rawDiscount) => {
    if (!rawDiscount) return {};
    if (typeof rawDiscount === 'string') {
      try {
        const parsed = JSON.parse(rawDiscount);
        return parsed && typeof parsed === 'object' ? parsed : {};
      } catch (e) {
        return {};
      }
    }
    return typeof rawDiscount === 'object' ? rawDiscount : {};
  };

  const convertDiscountToBonusMap = (discountMap) => {
    const converted = {};
    Object.keys(discountMap || {}).forEach((amountKey) => {
      // Legacy compatibility:
      // - Old deployments might only have amount_discount configured.
      // - If values look like "bonus rates" (e.g. 0.05, 0.4), treat them as bonus directly.
      // - If values look like "discount multipliers" (e.g. 0.95), do NOT convert to bonus.
      const v = Number(discountMap[amountKey]);
      if (!Number.isFinite(v) || v <= 0) return;
      if (v <= 0.5) {
        converted[Number(amountKey)] = Number(v.toFixed(4));
      }
    });
    return converted;
  };

  const fetchPresetPayAmountMap = async (presets) => {
    const amounts = Array.from(
      new Set(
        (presets || [])
          .map((item) => Number(item?.value))
          .filter((v) => Number.isFinite(v) && v > 0),
      ),
    );
    if (amounts.length === 0) {
      setPresetPayAmountMap({});
      return;
    }

    const entries = await Promise.all(
      amounts.map(async (amountValue) => {
        try {
          const res = await API.post('/api/user/amount', {
            amount: parseFloat(amountValue),
          });
          if (res?.data?.message === 'success') {
            const value = Number(res.data.data);
            return [amountValue, Number.isFinite(value) ? value : null];
          }
        } catch (e) {
          // ignore individual preset amount failures
        }
        return [amountValue, null];
      }),
    );

    const nextMap = {};
    entries.forEach(([amountValue, payAmount]) => {
      if (Number.isFinite(payAmount)) {
        nextMap[amountValue] = payAmount;
      }
    });
    setPresetPayAmountMap(nextMap);
  };

  const topUp = async () => {
    if (redemptionCode === '') {
      showInfo(t('请输入兑换码！'));
      return;
    }
    setIsSubmitting(true);
    try {
      const res = await API.post('/api/user/topup', {
        key: redemptionCode,
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        Modal.success({
          title: t('兑换成功！'),
          content: t('成功兑换额度：') + renderQuota(data),
          centered: true,
        });
        if (userState.user) {
          const updatedUser = {
            ...userState.user,
            quota: userState.user.quota + data,
          };
          userDispatch({ type: 'login', payload: updatedUser });
        }
        setRedemptionCode('');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(t('请求失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTopUpLink = () => {
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const preTopUp = async (payment) => {
    if (payment === 'stripe') {
      if (!enableStripeTopUp) {
        showError(t('管理员未开启Stripe充值！'));
        return;
      }
    } else {
      if (!enableOnlineTopUp) {
        showError(t('管理员未开启在线充值！'));
        return;
      }
    }

    setPayWay(payment);
    setPaymentLoading(true);
    try {
      if (payment === 'stripe') {
        await getStripeAmount();
      } else {
        await getAmount();
      }

      if (topUpCount < minTopUp) {
        showError(t('充值数量不能小于') + minTopUp);
        return;
      }
      setOpen(true);
    } catch (error) {
      showError(t('获取金额失败'));
    } finally {
      setPaymentLoading(false);
    }
  };

  const onlineTopUp = async () => {
    if (payWay === 'stripe') {
      // Stripe 支付处理
      if (amount === 0) {
        await getStripeAmount();
      }
    } else if (payWay === 'lantu') {
      // 蓝兔支付处理
      if (amount === 0) {
        await getAmount();
      }
    } else {
      // 普通支付处理
      if (amount === 0) {
        await getAmount();
      }
    }

    if (topUpCount < minTopUp) {
      showError('充值数量不能小于' + minTopUp);
      return;
    }
    setConfirmLoading(true);
    try {
      let res;
      if (payWay === 'stripe') {
        // Stripe 支付请求
        res = await API.post('/api/user/stripe/pay', {
          amount: parseInt(topUpCount),
          payment_method: 'stripe',
        });
      } else if (payWay === 'lantu') {
        // 蓝兔支付请求
        res = await API.post('/api/user/lantu/pay', {
          amount: parseInt(topUpCount),
          payment_method: 'lantu',
          client: /MicroMessenger|Android|iPhone|iPod|iPad|Windows Phone|Mobile/i.test(
            navigator.userAgent || '',
          )
            ? 'h5'
            : 'native',
        });
      } else {
        // 普通支付请求
        res = await API.post('/api/user/pay', {
          amount: parseInt(topUpCount),
          payment_method: payWay,
        });
      }

      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          if (payWay === 'stripe') {
            // Stripe 支付回调处理
            window.open(data.pay_link, '_blank');
          } else if (payWay === 'lantu') {
            // 蓝兔支付：移动端 H5 直接跳转；桌面端展示二维码并轮询订单状态
            if (data?.client === 'h5') {
              window.location.href = data.pay_link;
            } else if (data?.client === 'native') {
              setLantuPayUrl(data.pay_link || '');
              setLantuPayUrlKind(data.pay_link_kind || '');
              setLantuTradeNo(data.trade_no || '');
              setLantuPayModalOpen(true);
              setLantuPolling(true);
            } else {
              // Backward compatible fallback
              window.open(data.pay_link, '_blank');
            }
          } else {
            // 普通支付表单提交
            let params = data;
            let url = res.data.url;
            let form = document.createElement('form');
            form.action = url;
            form.method = 'POST';
            let isSafari =
              navigator.userAgent.indexOf('Safari') > -1 &&
              navigator.userAgent.indexOf('Chrome') < 1;
            if (!isSafari) {
              form.target = '_blank';
            }
            for (let key in params) {
              let input = document.createElement('input');
              input.type = 'hidden';
              input.name = key;
              input.value = params[key];
              form.appendChild(input);
            }
            document.body.appendChild(form);
            form.submit();
            document.body.removeChild(form);
          }
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
      showError(t('支付请求失败'));
    } finally {
      setOpen(false);
      setConfirmLoading(false);
    }
  };

  const creemPreTopUp = async (product) => {
    if (!enableCreemTopUp) {
      showError(t('管理员未开启 Creem 充值！'));
      return;
    }
    setSelectedCreemProduct(product);
    setCreemOpen(true);
  };

  const onlineCreemTopUp = async () => {
    if (!selectedCreemProduct) {
      showError(t('请选择产品'));
      return;
    }
    // Validate product has required fields
    if (!selectedCreemProduct.productId) {
      showError(t('产品配置错误，请联系管理员'));
      return;
    }
    setConfirmLoading(true);
    try {
      const res = await API.post('/api/user/creem/pay', {
        product_id: selectedCreemProduct.productId,
        payment_method: 'creem',
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          processCreemCallback(data);
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
      showError(t('支付请求失败'));
    } finally {
      setCreemOpen(false);
      setConfirmLoading(false);
    }
  };

  const processCreemCallback = (data) => {
    // 与 Stripe 保持一致的实现方式
    window.open(data.checkout_url, '_blank');
  };

  const getUserQuota = async () => {
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
    } else {
      showError(message);
    }
  };

  const getSubscriptionPlans = async () => {
    setSubscriptionLoading(true);
    try {
      const res = await API.get('/api/subscription/plans');
      if (res.data?.success) {
        setSubscriptionPlans(res.data.data || []);
      }
    } catch (e) {
      setSubscriptionPlans([]);
    } finally {
      setSubscriptionLoading(false);
    }
  };

  const getSubscriptionSelf = async () => {
    try {
      const res = await API.get('/api/subscription/self');
      if (res.data?.success) {
        setBillingPreference(
          res.data.data?.billing_preference || 'subscription_first',
        );
        // Active subscriptions
        const activeSubs = res.data.data?.subscriptions || [];
        setActiveSubscriptions(activeSubs);
        // All subscriptions (including expired)
        const allSubs = res.data.data?.all_subscriptions || [];
        setAllSubscriptions(allSubs);
      }
    } catch (e) {
      // ignore
    }
  };

  const updateBillingPreference = async (pref) => {
    const previousPref = billingPreference;
    setBillingPreference(pref);
    try {
      const res = await API.put('/api/subscription/self/preference', {
        billing_preference: pref,
      });
      if (res.data?.success) {
        showSuccess(t('更新成功'));
        const normalizedPref =
          res.data?.data?.billing_preference || pref || previousPref;
        setBillingPreference(normalizedPref);
      } else {
        showError(res.data?.message || t('更新失败'));
        setBillingPreference(previousPref);
      }
    } catch (e) {
      showError(t('请求失败'));
      setBillingPreference(previousPref);
    }
  };

  // 获取充值配置信息
  const getTopupInfo = async () => {
    try {
      const res = await API.get('/api/user/topup/info');
      const { message, data, success } = res.data;
      if (success) {
        const parsedDiscount = parseDiscountMap(data.discount || {});
        let parsedBonus = parseBonusMap(data.bonus || {});
        if (Object.keys(parsedBonus).length === 0) {
          parsedBonus = convertDiscountToBonusMap(parsedDiscount);
        }
        setTopupInfo({
          amount_options: data.amount_options || [],
          bonus: parsedBonus || {},
          discount: parsedDiscount || {},
        });
        const ratioNum = Number(data.topup_group_ratio);
        setTopupGroupRatio(
          Number.isFinite(ratioNum) && ratioNum > 0 ? ratioNum : 1,
        );

        // 处理支付方式
        let payMethods = data.pay_methods || [];
        try {
          if (typeof payMethods === 'string') {
            payMethods = JSON.parse(payMethods);
          }
          if (payMethods && payMethods.length > 0) {
            // 检查name和type是否为空
            payMethods = payMethods.filter((method) => {
              return method.name && method.type;
            });
            // 如果没有color，则设置默认颜色
            payMethods = payMethods.map((method) => {
              // 规范化最小充值数
              const normalizedMinTopup = Number(method.min_topup);
              method.min_topup = Number.isFinite(normalizedMinTopup)
                ? normalizedMinTopup
                : 0;

              // Stripe 的最小充值从后端字段回填
              if (
                method.type === 'stripe' &&
                (!method.min_topup || method.min_topup <= 0)
              ) {
                const stripeMin = Number(data.stripe_min_topup);
                if (Number.isFinite(stripeMin)) {
                  method.min_topup = stripeMin;
                }
              }

              if (!method.color) {
                if (method.type === 'alipay') {
                  method.color = 'rgba(var(--semi-blue-5), 1)';
                } else if (method.type === 'wxpay') {
                  method.color = 'rgba(var(--semi-green-5), 1)';
                } else if (method.type === 'lantu') {
                  method.color = 'rgba(var(--semi-green-5), 1)';
                } else if (method.type === 'stripe') {
                  method.color = 'rgba(var(--semi-purple-5), 1)';
                } else {
                  method.color = 'rgba(var(--semi-primary-5), 1)';
                }
              }
              return method;
            });
          } else {
            payMethods = [];
          }

          // 如果启用了 Stripe 支付，添加到支付方法列表
          // 这个逻辑现在由后端处理，如果 Stripe 启用，后端会在 pay_methods 中包含它

          setPayMethods(payMethods);
          const enableStripeTopUp = data.enable_stripe_topup || false;
          const enableOnlineTopUp = data.enable_online_topup || false;
          const enableCreemTopUp = data.enable_creem_topup || false;
          const minTopUpValue = enableOnlineTopUp
            ? data.min_topup
            : enableStripeTopUp
              ? data.stripe_min_topup
              : 1;
          setEnableOnlineTopUp(enableOnlineTopUp);
          setEnableStripeTopUp(enableStripeTopUp);
          setEnableCreemTopUp(enableCreemTopUp);
          setMinTopUp(minTopUpValue);
          setTopUpCount(minTopUpValue);

          // 设置 Creem 产品
          try {
            const products = JSON.parse(data.creem_products || '[]');
            setCreemProducts(products);
          } catch (e) {
            setCreemProducts([]);
          }

          // 预设充值档位：
          // 1) 优先后端自定义 amount_options
          // 2) 否则使用 0.8.1 版本的稳定档位（过滤掉小于最小充值的项）
          // 3) 若过滤后为空，再回退到按倍数生成
          const options =
            Array.isArray(data.amount_options) && data.amount_options.length > 0
              ? data.amount_options
              : LEGACY_PRESET_AMOUNTS;
          const normalized = options
            .map((n) => Number(n))
            .filter((n) => Number.isFinite(n) && n > 0)
            .filter((n) => n >= Number(minTopUpValue || 1));
          const fallback =
            normalized.length > 0 ? normalized : generatePresetAmounts(minTopUpValue).map((x) => x.value);
          const bonusMap = parsedBonus || {};
          const customPresets = fallback.map((amount) => ({
            value: amount,
            bonus: getBonusRateForAmount(bonusMap, amount),
          }));
          setPresetAmounts(customPresets);
          fetchPresetPayAmountMap(customPresets);

          // 尽量选中一个“稳定档位”，让展示更接近 0.8.1
          const initSelected =
            customPresets.find((p) => p.value === minTopUpValue) ||
            customPresets.find((p) => p.value >= minTopUpValue) ||
            null;
          const initialAmount = initSelected?.value || minTopUpValue;
          setTopUpCount(initialAmount);
          setSelectedPreset(initSelected ? initSelected.value : null);

          // 初始化显示实付金额
          getAmount(initialAmount);
        } catch (e) {
          console.log('解析支付方式失败:', e);
          setPayMethods([]);
        }
      } else {
        console.error('获取充值配置失败:', data);
      }
    } catch (error) {
      console.error('获取充值配置异常:', error);
    }
  };

  // 获取邀请链接
  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    const { success, message, data } = res.data;
    if (success) {
      let link = `${window.location.origin}/register?aff=${data}`;
      setAffLink(link);
    } else {
      showError(message);
    }
  };

  // 划转邀请额度
  const transfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
      return;
    }
    const res = await API.post(`/api/user/aff_transfer`, {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      getUserQuota().then();
    } else {
      showError(message);
    }
  };

  // 复制邀请链接
  const handleAffLinkClick = async () => {
    await copy(affLink);
    showSuccess(t('邀请链接已复制到剪切板'));
  };

  useEffect(() => {
    // 始终获取最新用户数据，确保余额等统计信息准确
    getUserQuota().then();
    setTransferAmount(getQuotaPerUnit());
  }, []);

  useEffect(() => {
    if (affFetchedRef.current) return;
    affFetchedRef.current = true;
    getAffLink().then();
  }, []);

  // 在 statusState 可用时获取充值信息
  useEffect(() => {
    getTopupInfo().then();
    getSubscriptionPlans().then();
    getSubscriptionSelf().then();
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      // const minTopUpValue = statusState.status.min_topup || 1;
      // setMinTopUp(minTopUpValue);
      // setTopUpCount(minTopUpValue);
      setTopUpLink(statusState.status.top_up_link || '');
      setPriceRatio(statusState.status.price || 1);

      setStatusLoading(false);
    }
  }, [statusState?.status]);

  const handleLantuPayCancel = () => {
    setLantuPayModalOpen(false);
    setLantuPolling(false);
    setLantuPayUrl('');
    setLantuPayUrlKind('');
    setLantuTradeNo('');
    setLantuCountdown(0);
  };

  // LanTu native pay countdown (5 minutes)
  useEffect(() => {
    if (!lantuPayModalOpen) return;
    setLantuCountdown(300);
    const timer = setInterval(() => {
      setLantuCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          setLantuPayModalOpen(false);
          setLantuPolling(false);
          showError(t('订单已过期'));
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [lantuPayModalOpen]);

  // LanTu native pay status polling
  useEffect(() => {
    let interval;
    if (!lantuPolling || !lantuPayModalOpen || !lantuTradeNo) return;
    interval = setInterval(async () => {
      try {
        const res = await API.get(
          `/api/user/lantu/status?trade_no=${encodeURIComponent(lantuTradeNo)}`,
        );
        const { message, status } = res.data || {};
        if (message !== 'success') {
          setLantuPayModalOpen(false);
          setLantuPolling(false);
          showError(t('获取订单状态失败'));
          return;
        }
        if (status === 'success') {
          setLantuPayModalOpen(false);
          setLantuPolling(false);
          showSuccess(t('充值成功'));
          getUserQuota().then();
          return;
        }
        if (status === 'expired') {
          setLantuPayModalOpen(false);
          setLantuPolling(false);
          showError(t('订单已过期'));
          return;
        }
        // pending -> keep polling
      } catch (e) {
        // keep polling; do not close modal on transient failures
      }
    }, 10000);
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [lantuPolling, lantuPayModalOpen, lantuTradeNo]);

  const renderAmount = () => {
    if (amount === 0) {
      return t('计算中...');
    }
    return amount + ' ' + t('元');
  };

  const getAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/amount', {
        amount: parseFloat(value),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    }
    setAmountLoading(false);
  };

  const getStripeAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/stripe/amount', {
        amount: parseFloat(value),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    } finally {
      setAmountLoading(false);
    }
  };

  const handleCancel = () => {
    setOpen(false);
  };

  const handleTransferCancel = () => {
    setOpenTransfer(false);
  };

  const handleOpenHistory = () => {
    setOpenHistory(true);
  };

  const handleHistoryCancel = () => {
    setOpenHistory(false);
  };

  const handleCreemCancel = () => {
    setCreemOpen(false);
    setSelectedCreemProduct(null);
  };

  // 选择预设充值额度
  const selectPresetAmount = (preset) => {
    setTopUpCount(preset.value);
    setSelectedPreset(preset.value);
    setAmount(0);
    // 触发金额计算以获取准确的支付金额（包含分组倍率等后端逻辑）
    getAmount(preset.value);
  };

  // 格式化大数字显示
  const formatLargeNumber = (num) => {
    return num.toString();
  };

  // 根据最小充值金额生成预设充值额度选项
  const generatePresetAmounts = (minAmount) => {
    const multipliers = [1, 5, 10, 30, 50, 100, 300, 500];
    return multipliers.map((multiplier) => ({
      value: minAmount * multiplier,
    }));
  };

  return (
    <div className='w-full max-w-7xl mx-auto relative min-h-screen lg:min-h-0 mt-[60px] px-2'>
      {/* 划转模态框 */}
      <TransferModal
        t={t}
        openTransfer={openTransfer}
        transfer={transfer}
        handleTransferCancel={handleTransferCancel}
        userState={userState}
        renderQuota={renderQuota}
        getQuotaPerUnit={getQuotaPerUnit}
        transferAmount={transferAmount}
        setTransferAmount={setTransferAmount}
      />

      {/* 充值确认模态框 */}
      <PaymentConfirmModal
        t={t}
        open={open}
        onlineTopUp={onlineTopUp}
        handleCancel={handleCancel}
        confirmLoading={confirmLoading}
        topUpCount={topUpCount}
        renderQuota={renderQuota}
        amountLoading={amountLoading}
        renderAmount={renderAmount}
        payWay={payWay}
        payMethods={payMethods}
        bonusRate={getBonusRateForAmount(topupInfo?.bonus, topUpCount)}
        groupRatio={topupGroupRatio}
      />

      {/* 蓝兔支付二维码模态框（桌面端 native） */}
      <Modal
        title={t('微信扫码支付')}
        visible={lantuPayModalOpen}
        onCancel={handleLantuPayCancel}
        maskClosable={false}
        centered
        footer={null}
      >
        <div className='flex flex-col items-center gap-3'>
          <div className='text-sm text-slate-600 dark:text-slate-300'>
            {t('请使用微信扫码完成支付')} · {t('剩余')} {lantuCountdown}s
          </div>
          {lantuPayUrl ? (
            lantuPayUrlKind === 'qr_text' ||
            (!lantuPayUrlKind &&
              /^weixin:\/\//i.test(String(lantuPayUrl || ''))) ? (
              <QRCodeSVG value={lantuPayUrl} size={220} />
            ) : (
              <img
                src={lantuPayUrl}
                alt='qrcode'
                style={{
                  width: 220,
                  height: 220,
                  borderRadius: 12,
                  border: '1px solid rgba(148, 163, 184, 0.35)',
                }}
              />
            )
          ) : (
            <div className='text-sm text-slate-600 dark:text-slate-300'>
              {t('二维码加载中...')}
            </div>
          )}
          {lantuTradeNo ? (
            <div className='text-xs text-slate-500 dark:text-slate-400'>
              {t('订单号')}：{lantuTradeNo}
            </div>
          ) : null}
        </div>
      </Modal>

      {/* 充值账单模态框 */}
      <TopupHistoryModal
        visible={openHistory}
        onCancel={handleHistoryCancel}
        t={t}
      />

      {/* Creem 充值确认模态框 */}
      <Modal
        title={t('确定要充值 $')}
        visible={creemOpen}
        onOk={onlineCreemTopUp}
        onCancel={handleCreemCancel}
        maskClosable={false}
        size='small'
        centered
        confirmLoading={confirmLoading}
      >
        {selectedCreemProduct && (
          <>
            <p>
              {t('产品名称')}：{selectedCreemProduct.name}
            </p>
            <p>
              {t('价格')}：{selectedCreemProduct.currency === 'EUR' ? '€' : '$'}
              {selectedCreemProduct.price}
            </p>
            <p>
              {t('充值额度')}：{selectedCreemProduct.quota}
            </p>
            <p>{t('是否确认充值？')}</p>
          </>
        )}
      </Modal>

      {/* 主布局区域 */}
      <div className='grid grid-cols-1 lg:grid-cols-12 gap-6'>
        <div className='lg:col-span-7'>
          <RechargeCard
            t={t}
            enableOnlineTopUp={enableOnlineTopUp}
            enableStripeTopUp={enableStripeTopUp}
            enableCreemTopUp={enableCreemTopUp}
            creemProducts={creemProducts}
            creemPreTopUp={creemPreTopUp}
            presetAmounts={presetAmounts}
            selectedPreset={selectedPreset}
            selectPresetAmount={selectPresetAmount}
            formatLargeNumber={formatLargeNumber}
            priceRatio={priceRatio}
            topUpCount={topUpCount}
            minTopUp={minTopUp}
            renderQuotaWithAmount={renderQuotaWithAmount}
            getAmount={getAmount}
            setTopUpCount={setTopUpCount}
            setSelectedPreset={setSelectedPreset}
            renderAmount={renderAmount}
            amountLoading={amountLoading}
            payMethods={payMethods}
            preTopUp={preTopUp}
            paymentLoading={paymentLoading}
            payWay={payWay}
            redemptionCode={redemptionCode}
            setRedemptionCode={setRedemptionCode}
            topUp={topUp}
            isSubmitting={isSubmitting}
            topUpLink={topUpLink}
            openTopUpLink={openTopUpLink}
            userState={userState}
            renderQuota={renderQuota}
            statusLoading={statusLoading}
            topupInfo={topupInfo}
            topupGroupRatio={topupGroupRatio}
            onOpenHistory={handleOpenHistory}
            subscriptionLoading={subscriptionLoading}
            subscriptionPlans={subscriptionPlans}
            billingPreference={billingPreference}
            onChangeBillingPreference={updateBillingPreference}
            activeSubscriptions={activeSubscriptions}
            allSubscriptions={allSubscriptions}
            reloadSubscriptionSelf={getSubscriptionSelf}
            presetPayAmountMap={presetPayAmountMap}
          />
        </div>
        <div className='lg:col-span-5'>
          <InvitationCard
            t={t}
            userState={userState}
            renderQuota={renderQuota}
            setOpenTransfer={setOpenTransfer}
            affLink={affLink}
            handleAffLinkClick={handleAffLinkClick}
          />
        </div>
      </div>
    </div>
  );
};

export default TopUp;
