<template>
  <body>
    <div class="container">
      <div class="logo">
        <img
          alt="Paul Graham profile logo"
          src="../assets/logo.png"
          v-bind:class="{ loading: this.$data.answerLoading }"
        />
      </div>
      <div class="body">
        <h1>Ask Paul Graham</h1>
        <div class="info">
          <it-button @click="showInfo = true">Info</it-button>
        </div>
        <it-modal v-model="showInfo">
          <template #header>
            <h2>Info</h2>
          </template>
          <template #body>
            <h3>About</h3>
            <p>
              <b>Ask Paul Graham</b> is a for-fun side project powered by
              <a href="https://openai.com/">OpenAI's GPT-3</a> and
              <a href="http://www.paulgraham.com/">Paul Graham's essays</a>.
              OpenAI capped costs at $360 per month which is currently being
              covered by the project maintainer so usage may get throttled
              depending on demand.
            </p>
            <h3>Questions</h3>
            <p>
              The <b>questions</b> feature answers user-provided questions using
              Graham's essays as training data. Note that these are answers
              <i>from GPT-3</i> and do not necessarily reflect Paul Graham's
              opinions.
            </p>
            <h3>Summaries</h3>
            <p>
              The <b>summaries</b> feature provides GPT-3-generated summaries of
              Graham's essays and may not necessarily reflect his summary of the
              given essay. Not all essays have been included due to length
              constraints on GPT-3.
            </p>
          </template>
        </it-modal>
        <it-alert
          type="warning"
          title="Warning"
          body='AskPaulGraham is no longer functioning due to the OpenAI deprecation of their "answers" feature. Feel free to open a pull request on the project repo to fix this.'
        />
        <br />
        <it-tabs box>
          <it-tab title="Questions">
            <form v-on:submit.prevent="submitForm">
              <div class="question">
                <it-input
                  placeholder="Ask your question"
                  v-model="question"
                  disabled
                />
              </div>
              <it-button disabled>Submit</it-button>
            </form>
            <div v-if="answer" class="answer">
              <it-alert
                type="success"
                show-icon="false"
                title="Answer"
                v-bind:body="answer"
              />
            </div>
          </it-tab>
          <it-tab title="Summaries">
            <div class="summaries">
              <it-collapse>
                <it-collapse-item
                  v-for="summary in summaries"
                  v-bind:key="summary.id"
                  v-bind:title="summary.title"
                >
                  <p>
                    {{ summary.summary }}
                  </p>
                  <a v-bind:href="summary.url">Link</a>
                </it-collapse-item>
              </it-collapse>
            </div>
          </it-tab>
        </it-tabs>
        <div class="links">
          <a href="https://www.buymeacoffee.com/forstmeier">Buy Me A Coffee</a>
          and
          <a href="https://github.com/forstmeier/askpaulgraham">GitHub</a>
        </div>
      </div>
    </div>
  </body>
</template>

<script>
import axios from "axios";

export default {
  name: "Main",
  data: function () {
    return {
      showInfo: false,
      question: "",
      answerLoading: false,
      answer: "",
      summaries: [],
      userID: "",
    };
  },
  methods: {
    submitForm() {
      this.$data.answerLoading = true;

      if (this.$data.question.length > 100) {
        this.$Message.danger({
          text: "Question must be 100 characters or less",
        });
        this.$data.answerLoading = false;
        return;
      }

      const body = {
        question: this.$data.question,
        user_id: this.$data.userID,
      };

      axios
        .post("/question", body)
        .then((response) => {
          if (response.data.answer === "") {
            this.$data.answer =
              "Sorry, I don't know the answer to that question.";
          } else {
            this.$data.answer = response.data.answer;
          }
        })
        .catch((error) => {
          if (error.response.status === 503) {
            this.$data.answer = "Sorry, I wasn't able to answer that question.";
          } else {
            this.$Message.danger({
              text: error.message,
            });
          }
        })
        .finally(() => {
          this.$data.answerLoading = false;
        });
    },
  },
  created: function () {
    axios.get("https://api.ipify.org?format=json").then((response) => {
      this.$data.userID = response.data.ip;
    });

    // axios
    //   .get("/summaries")
    //   .then((response) => {
    //     this.$data.summaries = response.data.summaries.sort(
    //       (a, b) => b.number - a.number
    //     );
    //   })
    //   .catch((error) => {
    //     this.$Message.danger({
    //       text: error.message,
    //     });
    //   });
  },
};
</script>

<style scoped>
body {
  height: 100%;
  width: 100%;
}

.container {
  height: 100%;
  margin: auto;
  padding-top: 6rem;
  width: 34rem;
}

.logo {
  text-align: center;
}

.body {
  padding-top: 2rem;
}

img {
  border-radius: 50%;
  filter: grayscale(100%);
}

.loading {
  animation: rotation 1s infinite linear;
  filter: grayscale(100%);
}

h1 {
  font-size: 3rem;
  text-align: center;
  padding-bottom: 1rem;
}

form,
.summaries {
  padding: 1rem;
}

.info,
.question,
h3,
p {
  padding-bottom: 1rem;
}

.answer {
  padding: 0rem 1rem 1rem 1rem;
}

.links {
  padding-top: 1rem;
  padding-bottom: 5rem;
}

@keyframes rotation {
  from {
    transform: rotate(359deg);
  }
  to {
    transform: rotate(0deg);
  }
}

@media screen and (max-width: 580px) {
  .container {
    padding: 1rem;
    width: 100%;
  }

  img {
    width: 50%;
  }

  h1 {
    font-size: 2rem;
  }
}
</style>