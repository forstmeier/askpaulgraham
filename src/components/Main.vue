<template>
  <body>
    <div class="container">
      <div class="logo">
        <img alt="Vue logo" src="../assets/logo.png" />
      </div>
      <div class="body">
        <h1>Ask Paul Graham</h1>
        <it-button @click="defaultModal = true">Info</it-button>
        <it-modal v-model="defaultModal">
          <template #header>
            <h2>Info</h2>
          </template>
          <template #body>
            <h3>About</h3>
            <p>
              <b>Ask Paul Graham</b> is a for-fun side project powered by GPT-3
              from <a href="https://openai.com/">OpenAI</a> and
              <a href="http://www.paulgraham.com/">Paul Graham's essays</a>.
              OpenAI caps usage at $120 per month which is currently being
              covered by the project maintainer so usage may get throttled
              depending on demand.
            </p>
            <h3>Questions</h3>
            <p>
              The <b>question</b> feature answers user-provided questions using
              Graham's essays as training data. Note that these are answers
              <i>from GPT-3</i> and do not necessarily reflect Paul Graham's
              opinions.
            </p>
            <h3>Summaries</h3>
            <p>
              The <b>summary</b> feature provides GPT-3-generated summaries of
              Graham's essays and may not necessarily reflect his summary of the
              given essay. Not all essays have been included due to length
              constraints on GPT-3.
            </p>
          </template>
        </it-modal>
      </div>
    </div>
  </body>
</template>



<script>
export default {
  name: "Main",
  data: function () {
    return {
      defaultModal: false,
    };
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

img {
  border-radius: 50%;
}

h3 {
  padding-bottom: 1rem;
}

p {
  padding-bottom: 1rem;
}
</style>