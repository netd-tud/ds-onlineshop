#!/usr/bin/python
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import math
import os
import random
import time
import datetime
from locust import FastHttpUser, TaskSet
from locust.exception import StopUser
from faker import Faker

fake = Faker()

products = [
    '0PUK6V6EV0',
    '1YMWWN1N4O',
    '2ZYFJ3GM2N',
    '66VCHSJNUP',
    '6E92ZMYYFZ',
    '9SIQT8TOJO',
    'L9ECAV7KIM',
    'LS4PSXUNUM',
    'OLJCESPC7Z']

BASE_PRODUCT_WEIGHTS = [15, 12, 11, 10, 10, 9, 8, 7, 5]

TREND_CYCLE_SECONDS = int(os.environ.get("TREND_CYCLE_MINUTES", 6 * 60)) * 60
TREND_BOOST = 10.0

def current_weights():
    n_products = len(products)
    total_cycle_time = n_products * TREND_CYCLE_SECONDS

    current_phase = (time.time() % total_cycle_time) / TREND_CYCLE_SECONDS

    weights = list(BASE_PRODUCT_WEIGHTS)

    rise_duration = 0.85
    fall_duration = 0.15

    for i in range(n_products):
        diff = current_phase - i

        half_n = n_products / 2.0
        diff = (diff + half_n) % n_products - half_n

        normalized_dist = 1.0

        if -rise_duration <= diff <= 0:
            normalized_dist = abs(diff) / rise_duration
        elif 0 < diff <= fall_duration:
            normalized_dist = diff / fall_duration

        if normalized_dist < 1.0:
            smooth_factor = (math.cos(normalized_dist * math.pi) + 1.0) / 2.0
            dynamic_boost = 1.0 + (TREND_BOOST - 1.0) * smooth_factor
            weights[i] *= dynamic_boost

    return weights

def pick_product():
    return random.choices(products, weights=current_weights())[0]

REGIONS = [
    {"currency": "USD", "utc_offset": -5, "population": 30},
    {"currency": "EUR", "utc_offset": 1,  "population": 25},
    {"currency": "GBP", "utc_offset": 0,  "population": 10},
    {"currency": "JPY", "utc_offset": 9,  "population": 20},
    {"currency": "CAD", "utc_offset": -8, "population": 5},
    {"currency": "TRY", "utc_offset": 3,  "population": 10},
]

def local_hour_now(utc_offset_hours):
    now = datetime.datetime.now(datetime.UTC)
    utc_hour = now.hour + now.minute / 60 + now.second / 3600
    return (utc_hour + utc_offset_hours) % 24

def regional_activity(region, peak_hour=18.0, spread=6.0):
    hour = local_hour_now(region["utc_offset"])
    delta = min(abs(hour - peak_hour), 24 - abs(hour - peak_hour))

    baseline = 0.10
    if delta >= spread:
        return baseline

    return max(baseline, math.cos((delta / spread) * (math.pi / 2)))

def index(l):
    l.client.get("/")

def setCurrency(l):
    l.client.post("/setCurrency", {'currency_code': l.region["currency"]})

def browseProduct(l):
    product = pick_product()
    l.client.get("/product/" + product)
    l.last_viewed_product = product

def addToCart(l):
    product = l.last_viewed_product if l.last_viewed_product and random.random() < 0.3 else pick_product()

    current_activity = regional_activity(l.region)

    # Base purchase of 1-2 items, plus up to 8 additional items during peak hours
    dynamic_quantity = random.randint(1, 2) + int(8 * current_activity)

    l.client.get("/product/" + product)
    l.client.post("/cart", {
        'product_id': product,
        'quantity': dynamic_quantity})
    l.cart_has_items = True

def viewCart(l):
    l.client.get("/cart")

def empty_cart(l):
    l.client.post('/cart/empty')
    l.cart_has_items = False

def checkout(l):
    if not getattr(l, "cart_has_items", False):
        addToCart(l)
    current_year = datetime.datetime.now().year + 1
    l.client.post("/cart/checkout", {
        'email': fake.email(),
        'street_address': fake.street_address(),
        'zip_code': fake.zipcode(),
        'city': fake.city(),
        'state': fake.state_abbr(),
        'country': fake.country(),
        'credit_card_number': fake.credit_card_number(card_type="visa"),
        'credit_card_expiration_month': random.randint(1, 12),
        'credit_card_expiration_year': random.randint(current_year, current_year + 70),
        'credit_card_cvv': f"{random.randint(100, 999)}",
    })
    l.cart_has_items = False

def logout(l):
    l.client.get('/logout')
    l.on_start()


class UserBehavior(TaskSet):

    def on_start(self):
        print("started")
        dynamic_weights = [r["population"] * regional_activity(r) for r in REGIONS]
        self.region = random.choices(REGIONS, weights=dynamic_weights)[0]
        self.last_viewed_product = None
        self.cart_has_items = False
        index(self)
        setCurrency(self)

    def wait_time(self):
        activity = max(regional_activity(self.region), 0.1)
        return random.uniform(1, 10) / activity

    tasks = {index: 1,
             setCurrency: 1,
             browseProduct: 10,
             addToCart: 3,
             viewCart: 3,
             checkout: 1,
             empty_cart: 1,
             logout: 1}


class WebsiteUser(FastHttpUser):
    tasks = [UserBehavior]

    default_headers = {
        "x-load-test": "true"
    }
